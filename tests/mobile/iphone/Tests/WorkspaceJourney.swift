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

    private func fill(_ id: String, _ text: String) throws {
        let field = app.descendants(matching: .any).matching(identifier: id).firstMatch
        XCTAssertTrue(field.waitForExistence(timeout: 15), "Missing field " + id)
        field.tap()
        app.typeText(text)
    }

    func testSchoolWorkList() throws {
        continueAfterFailure = false
        // The preceding native sign-in populated Keychain. A cold launch must
        // restore it through the server before the directory can be exercised.
        app.launch()
        let directory = try showing(["School modules"])
        try save(directory, "directory")
        if let loaded = try? showing(["Schools"], timeout: 10) { try save(loaded, "directory") }
        let schools = app.buttons.matching(identifier: "module-tenancy").firstMatch
        if schools.waitForExistence(timeout: 10) && schools.isHittable {
            schools.tap()
        } else {
            try tapText("Schools")
        }
        try fill("entry-title", "Campus review\n")
        try tapText("Add entry")
        if (try? showing(["Campus review", "Open"], timeout: 5)) == nil { app.swipeUp() }
        try save(showing(["Campus review", "Open"]), "workspace")
        if (try? showing(["Sign out"], timeout: 2)) == nil { app.swipeDown() }
        try tapText("Sign out")
        _ = try showing(["School ID", "Show password"])
        app.terminate()
        app.launch()
        _ = try showing(["School ID", "Show password"])
    }
}
