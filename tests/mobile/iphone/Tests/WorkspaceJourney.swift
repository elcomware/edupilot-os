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

    // Tap the painted button label. The heading uses the same text, so its
    // midpoint must also be inside the accessible button's own frame.
    private func tapSignIn() throws {
        let button = app.buttons["sign-in"]
        for _ in 0..<4 {
            if button.isHittable { break }
            app.scrollViews.firstMatch.swipeUp()
        }
        XCTAssertTrue(button.isEnabled && button.isHittable, "Sign in button unavailable")
        let screenshot = XCUIScreen.main.screenshot()
        let request = VNRecognizeTextRequest()
        request.recognitionLevel = .accurate
        try VNImageRequestHandler(cgImage: screenshot.image.cgImage!).perform([request])
        let bounds = app.frame
        for result in request.results ?? [] {
            guard result.topCandidates(1).first?.string.lowercased() == "sign in" else { continue }
            let label = result.boundingBox
            let point = CGPoint(x: bounds.width * label.midX, y: bounds.height * (1 - label.midY))
            if button.frame.contains(point) {
                app.coordinate(withNormalizedOffset: .zero).withOffset(CGVector(dx: point.x, dy: point.y)).tap()
                return
            }
        }
        XCTFail("No painted Sign in label inside the button")
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
        try tapSignIn()
        XCTAssertTrue(app.descendants(matching: .any)["login-error-incomplete-school"].waitForExistence(timeout: 8),
                      "Empty sign-in did not show field validation")
        _ = try enter("school-id", input.school)
        _ = try enter("email", input.email)
        _ = try enter("password", input.password, secure: true)
        // Drag inside the visible form, above the keyboard. A button whose
        // frame intersects the keyboard can still report itself as hittable.
        let scroll = app.scrollViews.firstMatch
        let keyboard = app.keyboards.firstMatch
        if keyboard.exists {
            let area = scroll.frame.intersection(app.frame)
            let bottom = min(area.maxY, keyboard.frame.minY) - 12
            let origin = app.coordinate(withNormalizedOffset: .zero)
            let start = origin.withOffset(CGVector(dx: area.midX, dy: bottom))
            let end = origin.withOffset(CGVector(dx: area.midX, dy: area.minY + 16))
            start.press(forDuration: 0.1, thenDragTo: end)
            let hidden = XCTNSPredicateExpectation(predicate: NSPredicate(format: "exists == false"), object: keyboard)
            XCTAssertEqual(XCTWaiter.wait(for: [hidden], timeout: 10), .completed, "Login keyboard still visible")
        }
        let field = app.descendants(matching: .any)
        XCTAssertFalse(field["login-error-incomplete-school"].exists, "Credential edits did not clear validation")
        XCTAssertTrue(field["school-id"].value as? String == input.school, "School changed after focus left")
        XCTAssertTrue(field["email"].value as? String == input.email, "Email changed after focus left")
        XCTAssertTrue((field["password"].value as? String)?.count == input.password.count, "Secure length changed after focus left")
        try tapSignIn()
        if !app.buttons["Sign out"].waitForExistence(timeout: 30) {
            let reasons = ["incomplete-school", "incomplete-email", "incomplete-password", "credentials", "network", "storage", "invalid", "forbidden", "expired", "conflict"]
            let reason = reasons.first { field["login-error-" + $0].exists }
            let system = XCUIApplication(bundleIdentifier: "com.apple.springboard")
            let fallback = system.alerts.firstMatch.exists ? "system-alert" : (field["sign-in"].exists ? "guest" : "unknown")
            XCTFail("Sign-in outcome: " + (reason ?? fallback))
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
