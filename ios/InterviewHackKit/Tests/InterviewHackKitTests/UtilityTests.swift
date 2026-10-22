// DesignDocument/11_TestDesign.md 11.4 / 5_FrontendDesign.md 5.9・5.10・5.12
import CoreGraphics
import CoreText
import Foundation
import Testing
@testable import InterviewHackKit

@Suite("資料の取り込み")
struct ExtractionTests {
    @Test("抽出テキストの後処理（改行・ゼロ幅文字・空行）")
    func normalize() {
        let raw = "  見出し\r\n\r\n\r\n\r\n本文\u{200B}です   \n\u{00A0}\nフッター "
        #expect(ExtractedTextNormalizer.normalize(raw) == "見出し\n\n本文です\n\nフッター")
    }

    @Test("テキストがほとんどない PDF は OCR に回す")
    func ocrHeuristic() {
        #expect(PDFTextExtractor.shouldUseOCR(text: " \n ", pageCount: 2))
        #expect(PDFTextExtractor.shouldUseOCR(text: String(repeating: "あ", count: 30), pageCount: 2))
        #expect(!PDFTextExtractor.shouldUseOCR(text: String(repeating: "あ", count: 40), pageCount: 2))
        #expect(!PDFTextExtractor.shouldUseOCR(text: "", pageCount: 0))
    }

    @Test("PDFKit でテキストのある PDF から本文を取り出す")
    func extractsPDFText() throws {
        let data = makePDF(text: "InterviewHack company research material text for extraction test")
        let result = try PDFTextExtractor.extract(data: data)
        #expect(result.pageCount == 1)
        #expect(result.text.contains("company research"))
        #expect(!result.needsOCR)
    }

    @Test("文字のない PDF は OCR が必要と判定し、ページを画像にできる")
    func blankPDF() throws {
        let data = makePDF(text: nil)
        let result = try PDFTextExtractor.extract(data: data)
        #expect(result.needsOCR)
        let images = try PDFTextExtractor.renderPages(data: data, scale: 1)
        #expect(images.count == 1)
    }

    @Test("PDF でないデータは読めない")
    func invalidPDF() {
        #expect(throws: PDFTextExtractor.ExtractionError.unreadable) {
            _ = try PDFTextExtractor.extract(data: Data("not a pdf".utf8))
        }
    }

    @Test("3,000 字を超える資料は要約を案内する（4.3.5）")
    func lengthAdvice() {
        #expect(ResearchLengthAdvice.evaluate(String(repeating: "a", count: 3_000)) == .ok)
        let advice = ResearchLengthAdvice.evaluate(String(repeating: "a", count: 3_001))
        #expect(advice == .suggestSummarize(length: 3_001))
        #expect(advice.message?.contains("8000") == true)
    }

    @Test("URL 取り込みの抽出スクリプトは article / main を優先する")
    func extractionScript() {
        #expect(WebExtraction.javaScript.contains("'article', 'main'"))
        #expect(WebExtraction.javaScript.contains("document.body.innerText"))
    }

    @Test("OCR の認識言語は日本語と英語")
    func ocrLanguages() {
        #expect(TextRecognizer.recognitionLanguages == ["ja-JP", "en-US"])
    }

    private func makePDF(text: String?) -> Data {
        let data = NSMutableData()
        var box = CGRect(x: 0, y: 0, width: 595, height: 842)
        let consumer = CGDataConsumer(data: data as CFMutableData)!
        let context = CGContext(consumer: consumer, mediaBox: &box, nil)!
        context.beginPDFPage(nil)
        if let text {
            let font = CTFontCreateWithName("Helvetica" as CFString, 14, nil)
            let attributed = NSAttributedString(string: text, attributes: [NSAttributedString.Key(kCTFontAttributeName as String): font])
            let line = CTLineCreateWithAttributedString(attributed)
            context.textPosition = CGPoint(x: 40, y: 700)
            CTLineDraw(line, context)
        }
        context.endPDFPage()
        context.closePDF()
        return data as Data
    }
}

@Suite("差分")
struct TextDiffTests {
    @Test("文字単位で挿入と削除を求める")
    func characterDiff() {
        let segments = TextDiff.diff(old: "私は結論から話す", new: "私は必ず結論から話します")
        #expect(segments == [.equal("私は"), .inserted("必ず"), .equal("結論から話"), .inserted("しま"), .equal("す")])
        #expect(TextDiff.changeCount(segments) == 4)
    }

    @Test("同じ文章は差分なし、空からは全挿入")
    func edgeCases() {
        #expect(TextDiff.diff(old: "同じ", new: "同じ") == [.equal("同じ")])
        #expect(TextDiff.diff(old: "", new: "新規") == [.inserted("新規")])
        #expect(TextDiff.diff(old: "削除", new: "") == [.deleted("削除")])
    }

    @Test("長い文章は行単位で比べる")
    func lineDiff() {
        let old = "a\nb\nc"
        let new = "a\nB\nc"
        let segments = TextDiff.diff(old: old, new: new, characterLimit: 1)
        #expect(segments == [.equal("a\n"), .deleted("b\n"), .inserted("B\n"), .equal("c\n")])
    }
}

@Suite("入力検査・設定・モデル")
struct MiscTests {
    @Test("面接ログの必須・文字数")
    func interviewLogValidation() throws {
        #expect(throws: FieldValidationError.required("質問")) {
            try Validator.interviewLog(InterviewLogInput(questionText: "  ", answerText: "回答"))
        }
        #expect(throws: FieldValidationError.tooLong("回答", max: 2_000)) {
            try Validator.interviewLog(InterviewLogInput(questionText: "Q", answerText: String(repeating: "a", count: 2_001)))
        }
        try Validator.interviewLog(InterviewLogInput(questionText: "Q", answerText: "A"))
    }

    @Test("取り込む URL は http / https のみ。スキームがなければ https を補う")
    func webURL() throws {
        #expect(try Validator.webURL("example.co.jp/recruit").absoluteString == "https://example.co.jp/recruit")
        #expect(throws: FieldValidationError.invalidURL) { _ = try Validator.webURL("file:///etc/passwd") }
        #expect(throws: FieldValidationError.invalidURL) { _ = try Validator.webURL("javascript:alert(1)") }
    }

    @Test("文字数制限の ±10%")
    func characterCount() {
        #expect(CharacterCount(text: String(repeating: "a", count: 360), limit: 400).isWithinTolerance)
        #expect(!CharacterCount(text: String(repeating: "a", count: 359), limit: 400).isWithinTolerance)
        #expect(CharacterCount(text: String(repeating: "a", count: 440), limit: 400).isWithinTolerance)
        #expect(CharacterCount(text: String(repeating: "a", count: 441), limit: 400).isOverLimit)
        #expect(CharacterCount(text: "abc", limit: nil).label == "3 字")
        #expect(CharacterCount(text: "abc", limit: 400).label == "3 / 400 字")
    }

    @Test("Info.plist の設定：HTTP は dev の localhost だけ許可する")
    func appConfig() throws {
        let dev = try AppConfig.load(from: ["API_BASE_URL": "http://localhost:8080", "APP_ENV": "dev"])
        #expect(dev.environment == .dev)
        let prod = try AppConfig.load(from: ["API_BASE_URL": "https://api.example.run.app", "APP_ENV": "prod"])
        #expect(prod.apiBaseURL.host == "api.example.run.app")
        #expect(throws: AppConfig.LoadError.invalid("API_BASE_URL")) {
            _ = try AppConfig.load(from: ["API_BASE_URL": "http://api.example.com", "APP_ENV": "prod"])
        }
        #expect(throws: AppConfig.LoadError.missing("APP_ENV")) {
            _ = try AppConfig.load(from: ["API_BASE_URL": "https://a.example.com"])
        }
    }

    @Test("日付（YYYY-MM-DD）の読み書きと比較")
    func calendarDate() throws {
        let date = try #require(CalendarDate("2026-11-01"))
        #expect(date.description == "2026-11-01")
        #expect(CalendarDate("2026-13-01") == nil)
        #expect(CalendarDate(year: 2026, month: 10, day: 31) < date)
        let encoded = try JSONCoding.makeEncoder().encode(["d": date])
        #expect(String(decoding: encoded, as: UTF8.self) == #"{"d":"2026-11-01"}"#)
    }

    @Test("日時は小数秒あり / なしの両方を読む")
    func dates() {
        #expect(JSONCoding.parseDate("2026-10-22T10:00:00Z") != nil)
        #expect(JSONCoding.parseDate("2026-10-22T10:00:00.123456Z") != nil)
        #expect(JSONCoding.parseDate("2026/10/22") == nil)
    }

    @Test("共有リンクの状態（公開中 / 期限切れ / 無効）")
    func shareLinkState() {
        let now = Date(timeIntervalSince1970: 1_000)
        var link = ESShareLink(id: "l1", sectionTitles: ["志望動機"], showCompanyName: false,
                             expiresAt: Date(timeIntervalSince1970: 2_000), revokedAt: nil, createdAt: nil, isActive: true)
        #expect(link.state(now: now) == .active)
        #expect(link.state(now: Date(timeIntervalSince1970: 2_000)) == .expired)
        link.revokedAt = Date(timeIntervalSince1970: 1_500)
        #expect(link.state(now: now) == .revoked)
        #expect(ShareLinkExpiry.default == .sevenDays)
        #expect(ShareLinkExpiry.allCases.map(\.rawValue) == [1, 7, 30])
    }

    @Test("未知のエラーコードも保持する")
    func unknownErrorCode() {
        #expect(APIErrorCode(rawValue: "SOMETHING_NEW") == .other("SOMETHING_NEW"))
        #expect(APIErrorCode(rawValue: "SHARE_LINK_GONE") == .shareLinkGone)
        #expect(APIErrorCode.featureDisabled.rawValue == "FEATURE_DISABLED")
    }

    @Test("research_summarize は資料 1 件が必須")
    func researchRequestValidation() {
        #expect(throws: PromptSessionRequest.ValidationError.invalidResearchMaterialCount) {
            try PromptSessionRequest(useCase: .researchSummarize, companyId: "c1", researchMaterialIds: ["r1", "r2"]).validate()
        }
        #expect(throws: PromptSessionRequest.ValidationError.missingTarget("interviewLogId")) {
            try PromptSessionRequest(useCase: .answerReview).validate()
        }
    }
}
