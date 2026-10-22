// DesignDocument/5_FrontendDesign.md 5.10 資料の取り込み（D11）
// 抽出は端末上で行い、テキストだけを BFF に送る。ファイル本体は保存しない。
import Foundation
#if canImport(PDFKit)
import PDFKit
#endif
#if canImport(Vision)
import Vision
#endif
import CoreGraphics

public enum WebExtraction {
    /// 5.10.1：`article` / `main` 要素があれば優先し、なければ `document.body.innerText`
    public static let javaScript = """
    (function() {
      var candidates = ['article', 'main', '[role="main"]'];
      for (var i = 0; i < candidates.length; i++) {
        var el = document.querySelector(candidates[i]);
        if (el && el.innerText && el.innerText.trim().length > 200) { return el.innerText; }
      }
      return document.body ? document.body.innerText : '';
    })();
    """
}

public enum ExtractedTextNormalizer {
    /// 抽出テキストの後処理：改行の統一、ゼロ幅文字の除去、行末の空白除去、3 行以上の空行を 1 行に
    public static func normalize(_ raw: String) -> String {
        var text = raw.replacingOccurrences(of: "\r\n", with: "\n").replacingOccurrences(of: "\r", with: "\n")
        for invisible in ["\u{200B}", "\u{200C}", "\u{200D}", "\u{FEFF}", "\u{00AD}"] {
            text = text.replacingOccurrences(of: invisible, with: "")
        }
        text = text.replacingOccurrences(of: "\u{00A0}", with: " ")
        let lines = text.components(separatedBy: "\n").map { line -> String in
            var line = line
            while let last = line.last, last == " " || last == "\t" { line.removeLast() }
            return line
        }
        var result: [String] = []
        var blankRun = 0
        for line in lines {
            if line.trimmingCharacters(in: .whitespaces).isEmpty {
                blankRun += 1
                if blankRun == 1 { result.append("") }
            } else {
                blankRun = 0
                result.append(line)
            }
        }
        return result.joined(separator: "\n").trimmingCharacters(in: .whitespacesAndNewlines)
    }
}

/// 5.10.4 / 4.3.5：長い資料の案内
public enum ResearchLengthAdvice: Equatable, Sendable {
    case ok
    /// プロンプトでは先頭 3,000 字に切り詰められる。先に要約すると使いやすい
    case suggestSummarize(length: Int)

    public static let promptExcerptLimit = 3_000
    public static let summarizeInputLimit = 8_000

    public static func evaluate(_ text: String) -> ResearchLengthAdvice {
        text.count > promptExcerptLimit ? .suggestSummarize(length: text.count) : .ok
    }

    public var message: String? {
        switch self {
        case .ok:
            return nil
        case .suggestSummarize(let length):
            return "本文が \(length) 字あります。プロンプトでは先頭 \(Self.promptExcerptLimit) 字までしか使われないため、"
                + "先に要約（AI連携）すると使いやすくなります。要約では先頭 \(Self.summarizeInputLimit) 字までを使います"
        }
    }
}

public struct PDFExtractionResult: Equatable, Sendable {
    public var text: String
    public var pageCount: Int
    /// テキストがほとんど取れない（スキャン PDF とみなし、OCR に回す。5.10.2）
    public var needsOCR: Bool
}

public enum PDFTextExtractor {
    /// 1 ページあたりの文字数（空白を除く）がこれ未満ならスキャン PDF とみなす
    public static let minimumCharactersPerPage = 20

    public static func shouldUseOCR(text: String, pageCount: Int) -> Bool {
        guard pageCount > 0 else { return false }
        let visible = text.unicodeScalars.filter { !CharacterSet.whitespacesAndNewlines.contains($0) }.count
        return visible < minimumCharactersPerPage * pageCount
    }

    public enum ExtractionError: Error, Equatable {
        case unreadable
        case unsupported
    }

    #if canImport(PDFKit)
    public static func extract(data: Data) throws -> PDFExtractionResult {
        guard let document = PDFDocument(data: data) else { throw ExtractionError.unreadable }
        let pageCount = document.pageCount
        var pages: [String] = []
        for index in 0..<pageCount {
            if let page = document.page(at: index), let string = page.string {
                pages.append(string)
            }
        }
        let text = ExtractedTextNormalizer.normalize(pages.joined(separator: "\n\n"))
        return PDFExtractionResult(text: text, pageCount: pageCount,
                                   needsOCR: shouldUseOCR(text: text, pageCount: pageCount))
    }

    /// スキャン PDF の各ページを画像にする（OCR 用）
    public static func renderPages(data: Data, scale: CGFloat = 2) throws -> [CGImage] {
        guard let document = PDFDocument(data: data) else { throw ExtractionError.unreadable }
        var images: [CGImage] = []
        for index in 0..<document.pageCount {
            guard let page = document.page(at: index) else { continue }
            let bounds = page.bounds(for: .mediaBox)
            let width = Int(bounds.width * scale)
            let height = Int(bounds.height * scale)
            guard width > 0, height > 0,
                  let context = CGContext(data: nil, width: width, height: height, bitsPerComponent: 8, bytesPerRow: 0,
                                          space: CGColorSpaceCreateDeviceRGB(),
                                          bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue) else { continue }
            context.setFillColor(CGColor(red: 1, green: 1, blue: 1, alpha: 1))
            context.fill(CGRect(x: 0, y: 0, width: width, height: height))
            context.scaleBy(x: scale, y: scale)
            page.draw(with: .mediaBox, to: context)
            if let image = context.makeImage() { images.append(image) }
        }
        return images
    }
    #else
    public static func extract(data: Data) throws -> PDFExtractionResult { throw ExtractionError.unsupported }
    #endif
}

public enum TextRecognizer {
    /// 5.10.3：`VNRecognizeTextRequest`。認識言語は ja-JP と en-US、精度は accurate
    public static let recognitionLanguages = ["ja-JP", "en-US"]

    #if canImport(Vision)
    public static func recognize(_ image: CGImage) async throws -> String {
        try await withCheckedThrowingContinuation { continuation in
            let request = VNRecognizeTextRequest { request, error in
                if let error {
                    continuation.resume(throwing: error)
                    return
                }
                let observations = (request.results as? [VNRecognizedTextObservation]) ?? []
                let lines = observations.compactMap { $0.topCandidates(1).first?.string }
                continuation.resume(returning: lines.joined(separator: "\n"))
            }
            request.recognitionLevel = .accurate
            request.recognitionLanguages = recognitionLanguages
            request.usesLanguageCorrection = true
            let handler = VNImageRequestHandler(cgImage: image, options: [:])
            do {
                try handler.perform([request])
            } catch {
                continuation.resume(throwing: error)
            }
        }
    }

    public static func recognize(pages: [CGImage]) async throws -> String {
        var texts: [String] = []
        for page in pages {
            texts.append(try await recognize(page))
        }
        return ExtractedTextNormalizer.normalize(texts.joined(separator: "\n\n"))
    }
    #endif
}
