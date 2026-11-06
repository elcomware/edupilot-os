import XCTest
import Vision

@MainActor
final class WorkspaceJourney: XCTestCase {
    private let app = XCUIApplication(bundleIdentifier: "org.edupilot.mobile")

    private struct Frame {
        let screenshot: XCUIScreenshot
        let labels: [String: CGRect]
    }

    private func frame() throws -> Frame {
        let screenshot = XCUIScreen.main.screenshot()
        let request = VNRecognizeTextRequest()
        request.recognitionLevel = .accurate
        request.recognitionLanguages = ["en-US"]
        request.usesLanguageCorrection = false
        try VNImageRequestHandler(cgImage: screenshot.image.cgImage!).perform([request])
        var labels: [String: CGRect] = [:]
        for observation in request.results ?? [] {
            if let text = observation.topCandidates(1).first?.string {
                labels[text.lowercased()] = observation.boundingBox
            }
        }
        return Frame(screenshot: screenshot, labels: labels)
    }

    // Verify the painted frame and use the same pixels for evidence or taps.
    private func showing(_ labels: [String], timeout: TimeInterval = 35) throws -> Frame {
        let deadline = Date().addingTimeInterval(timeout)
        repeat {
            let screen = try frame()
            if labels.allSatisfy({ screen.labels[$0.lowercased()] != nil }) { return screen }
            Thread.sleep(forTimeInterval: 0.35)
        } while Date() < deadline
        throw NSError(domain: "MobileProof", code: 1, userInfo: [
            NSLocalizedDescriptionKey: "Timed out waiting for visible labels: " + labels.joined(separator: ", ")
        ])
    }

    private func tapText(_ text: String) throws {
        let rect = try showing([text]).labels[text.lowercased()]!
        app.coordinate(withNormalizedOffset: CGVector(dx: rect.midX, dy: 1 - rect.midY)).tap()
    }

    private func save(_ screen: Frame, _ name: String) throws {
        let documents = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
        try screen.screenshot.pngRepresentation.write(to: documents.appendingPathComponent("edupilot-" + name + ".png"))
    }

    private func enter(_ id: String, _ text: String, secure: Bool = false) throws -> XCUIElement {
        let field = app.descendants(matching: .any).matching(identifier: id).firstMatch
        XCTAssertTrue(field.waitForExistence(timeout: 15), "Missing field " + id)
        field.tap()
        XCTAssertTrue(app.keyboards.firstMatch.waitForExistence(timeout: 10), "Keyboard unavailable for " + id)
        // Synchronize hardware-key injection with the focused native field.
        // Never retry a character or replace the value after a mismatch.
        var prefix = ""
        for character in text {
            field.typeText(String(character))
            prefix.append(character)
            let expected = secure
                ? NSPredicate(format: "value MATCHES %@", ".{\(prefix.count)}")
                : NSPredicate(format: "value == %@", prefix)
            let received = XCTNSPredicateExpectation(predicate: expected, object: field)
            XCTAssertEqual(XCTWaiter.wait(for: [received], timeout: 3), .completed,
                           "Input diverged for " + id + " at character " + String(prefix.count))
        }
        return field
    }

    private func fill(_ id: String, _ text: String) throws {
        let field = try enter(id, text)
        XCTAssertEqual(field.value as? String, text, "Incorrect entry before submit")
        field.typeText("\n")
        let dismissed = NSPredicate(format: "exists == false")
        let hidden = XCTNSPredicateExpectation(predicate: dismissed, object: app.keyboards.firstMatch)
        XCTAssertEqual(XCTWaiter.wait(for: [hidden], timeout: 10), .completed, "Keyboard still covers the form")
    }

    func testSignIn() throws {
        continueAfterFailure = false
        struct Credentials: Decodable {
            let school: String
            let email: String
            let password: String
        }
        let url = try XCTUnwrap(Bundle(for: Self.self).url(forResource: "fixture", withExtension: "json"))
        let input = try JSONDecoder().decode(Credentials.self, from: Data(contentsOf: url))
        app.launch()
        try save(showing(["School ID", "Show password"]), "signin")
        _ = try enter("school-id", input.school)
        _ = try enter("email", input.email)
        _ = try enter("password", input.password, secure: true)
        let button = app.buttons["sign-in"]
        for _ in 0..<4 {
            if button.isHittable { break }
            app.scrollViews.firstMatch.swipeUp()
        }
        XCTAssertTrue(button.isHittable, "Sign in button unavailable")
        button.tap()
        if !app.buttons["Sign out"].waitForExistence(timeout: 30) {
            let reasons = [
                ("incomplete", "Enter a school ID, email and password."),
                ("credentials", "Check your school ID, email and password."),
                ("network", "The server could not complete this request. Check your connection and try again."),
                ("storage", "Your session could not be saved. Unlock your device and sign in again."),
            ]
            let reason = reasons.first { app.descendants(matching: .any).matching(
                NSPredicate(format: "label == %@", $0.1)).firstMatch.exists }?.0
            XCTFail("Sign-in outcome: " + (reason ?? (button.exists ? "guest" : "unknown")))
        }
    }

    func testSchoolWorkList() throws {
        continueAfterFailure = false
        // The preceding native sign-in populated Keychain. A cold launch must
        // restore it through the server before the directory can be exercised.
        app.launch()
        let directory = try showing(["School modules"])
        XCTAssertTrue(app.staticTexts["demo-ios"].waitForExistence(timeout: 10), "Wrong restored school")
        try save(directory, "directory")
        if let loaded = try? showing(["Schools"], timeout: 10) { try save(loaded, "directory") }
        let schools = app.buttons.matching(identifier: "module-tenancy").firstMatch
        if schools.waitForExistence(timeout: 10) && schools.isHittable {
            schools.tap()
        } else {
            try tapText("Schools")
        }
        try fill("entry-title", "Campus review")
        let add = app.buttons["Add entry"]
        XCTAssertTrue(add.waitForExistence(timeout: 10) && add.isHittable, "Add entry button unavailable")
        add.tap()
        XCTAssertTrue(app.staticTexts["Entry saved."].waitForExistence(timeout: 25), "Server save acknowledgement missing")
        for _ in 0..<4 {
            if (try? showing(["Campus review", "Open"], timeout: 2)) != nil { break }
            app.swipeUp()
        }
        try save(showing(["Campus review", "Open"]), "workspace")
        if (try? showing(["Sign out"], timeout: 2)) == nil { app.swipeDown() }
        try tapText("Sign out")
        _ = try showing(["School ID", "Show password"])
        app.terminate()
        app.launch()
        _ = try showing(["School ID", "Show password"])
    }
}
