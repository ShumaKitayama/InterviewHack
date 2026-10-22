// DesignDocument/3_UIUXDesign.md 3.5.15 URL取り込み（S-Data-06）/ 3.5.16 PDF・画像取り込み（S-Data-07）
// DesignDocument/5_FrontendDesign.md 5.10 資料の取り込み（端末上で抽出し、テキストだけを保存する。D11）
import InterviewHackKit
import PhotosUI
import SwiftUI
import UniformTypeIdentifiers
import WebKit

/// S-Data-06 URL取り込み
struct URLImportView: View {
    let companyId: ID
    let onSaved: () -> Void

    @ViewState private var urlText = ""
    @ViewState private var loadedURL: URL?
    @ViewState private var controller = WebPageController()
    @ViewState private var extracted: String?
    @ViewState private var error: Error?

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                TextField("https://", text: $urlText)
                    .urlInput()
                    .textFieldStyle(.roundedBorder)
                    .onSubmit(open)
                Button("開く", action: open)
            }
            .padding()
            if let error {
                ErrorBanner(error: error).padding(.horizontal)
            }
            if let loadedURL {
                WebPageView(url: loadedURL, controller: controller)
                Button {
                    Task {
                        do {
                            extracted = try await controller.extractText()
                        } catch {
                            self.error = error
                        }
                    }
                } label: {
                    Label("本文を取り込む", systemImage: "text.viewfinder").frame(maxWidth: .infinity, minHeight: 44)
                }
                .buttonStyle(.borderedProminent)
                .padding()
            } else {
                Spacer()
                Text("取り込みはあなたの操作で 1 ページずつ行います。保存するのは抽出したテキストと URL だけです。")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                    .padding()
                Spacer()
            }
        }
        .navigationTitle("Webページの取り込み")
        .inlineNavigationTitle()
        .navigationDestination(item: Binding(get: { extracted.map(IdentifiedText.init) }, set: { extracted = $0?.text })) { item in
            MaterialTextEditorView(companyId: companyId, type: .url, sourceUrl: loadedURL?.absoluteString,
                                   initialText: item.text, onSaved: onSaved)
        }
    }

    private func open() {
        do {
            loadedURL = try Validator.webURL(urlText)
            error = nil
        } catch {
            self.error = error
        }
    }
}

struct IdentifiedText: Identifiable, Hashable {
    let text: String
    var id: String { text }
}

/// WKWebView の操作（本文の抽出）
@MainActor
final class WebPageController: NSObject, WKNavigationDelegate, WKUIDelegate {
    weak var webView: WKWebView?

    enum WebError: Error, LocalizedError {
        case notLoaded
        case empty

        var errorDescription: String? {
            switch self {
            case .notLoaded: return "ページの読み込みが終わっていません"
            case .empty: return "本文を取り出せませんでした"
            }
        }
    }

    static func makeWebView() -> WKWebView {
        let configuration = WKWebViewConfiguration()
        // 5.10.1：非永続のデータストアを使い、Cookie などを残さない
        configuration.websiteDataStore = .nonPersistent()
        return WKWebView(frame: .zero, configuration: configuration)
    }

    func extractText() async throws -> String {
        guard let webView else { throw WebError.notLoaded }
        let result = try await webView.evaluateJavaScript(WebExtraction.javaScript)
        let text = ExtractedTextNormalizer.normalize((result as? String) ?? "")
        guard !text.isEmpty else { throw WebError.empty }
        return text
    }

    // 5.13：ページ内のリンクで外部アプリを開かせない。ファイルのダウンロードも禁止する
    func webView(_ webView: WKWebView, decidePolicyFor navigationAction: WKNavigationAction,
                 decisionHandler: @escaping @MainActor (WKNavigationActionPolicy) -> Void) {
        let scheme = navigationAction.request.url?.scheme?.lowercased()
        if navigationAction.shouldPerformDownload || !(scheme == "http" || scheme == "https" || scheme == "about") {
            decisionHandler(.cancel)
        } else {
            decisionHandler(.allow)
        }
    }

    func webView(_ webView: WKWebView, decidePolicyFor navigationResponse: WKNavigationResponse,
                 decisionHandler: @escaping @MainActor (WKNavigationResponsePolicy) -> Void) {
        decisionHandler(navigationResponse.canShowMIMEType ? .allow : .cancel)
    }

    /// 新しいウィンドウを開くリンクは、同じ WebView では開かない
    func webView(_ webView: WKWebView, createWebViewWith configuration: WKWebViewConfiguration,
                 for navigationAction: WKNavigationAction, windowFeatures: WKWindowFeatures) -> WKWebView? {
        nil
    }
}

#if os(iOS)
struct WebPageView: UIViewRepresentable {
    let url: URL
    let controller: WebPageController

    func makeUIView(context: Context) -> WKWebView {
        let webView = WebPageController.makeWebView()
        webView.navigationDelegate = controller
        webView.uiDelegate = controller
        controller.webView = webView
        webView.load(URLRequest(url: url))
        return webView
    }

    func updateUIView(_ webView: WKWebView, context: Context) {
        if webView.url != url { webView.load(URLRequest(url: url)) }
    }
}
#else
struct WebPageView: NSViewRepresentable {
    let url: URL
    let controller: WebPageController

    func makeNSView(context: Context) -> WKWebView {
        let webView = WebPageController.makeWebView()
        webView.navigationDelegate = controller
        webView.uiDelegate = controller
        controller.webView = webView
        webView.load(URLRequest(url: url))
        return webView
    }

    func updateNSView(_ webView: WKWebView, context: Context) {
        if webView.url != url { webView.load(URLRequest(url: url)) }
    }
}
#endif

/// S-Data-07 PDF・画像取り込み
struct FileImportView: View {
    let companyId: ID
    let onSaved: () -> Void

    @ViewState private var pickingPDF = false
    @ViewState private var photoItem: PhotosPickerItem?
    @ViewState private var extracted: IdentifiedText?
    @ViewState private var extractedType: ResearchMaterialType = .pdf
    @ViewState private var isWorking = false
    @ViewState private var error: Error?
    #if os(iOS)
    @ViewState private var showingCamera = false
    #endif

    var body: some View {
        Form {
            Section {
                Text("ファイル本体はアップロードも保存もしません。端末上で取り出したテキストだけを保存します。")
                    .font(.footnote)
            }
            Section {
                Button { pickingPDF = true } label: { Label("ファイルから選ぶ（PDF）", systemImage: "doc") }
                PhotosPicker(selection: $photoItem, matching: .images) {
                    Label("写真から選ぶ", systemImage: "photo")
                }
                #if os(iOS)
                if UIImagePickerController.isSourceTypeAvailable(.camera) {
                    Button { showingCamera = true } label: { Label("カメラで撮る", systemImage: "camera") }
                }
                #endif
            }
            if isWorking {
                ProgressView("テキストを取り出しています")
            }
            if let error { ErrorBanner(error: error) }
        }
        .navigationTitle("PDF・画像の取り込み")
        .fileImporter(isPresented: $pickingPDF, allowedContentTypes: [.pdf]) { result in
            Task { await handlePDF(result) }
        }
        .onChange(of: photoItem) { _, item in
            guard let item else { return }
            Task { await handlePhoto(item) }
        }
        #if os(iOS)
        .sheet(isPresented: $showingCamera) {
            CameraPicker { image in
                Task { await handleImage(image) }
            }
        }
        #endif
        .navigationDestination(item: $extracted) { item in
            MaterialTextEditorView(companyId: companyId, type: extractedType, sourceUrl: nil, initialText: item.text, onSaved: onSaved)
        }
    }

    private func handlePDF(_ result: Result<URL, Error>) async {
        isWorking = true
        defer { isWorking = false }
        do {
            let url = try result.get()
            let accessing = url.startAccessingSecurityScopedResource()
            defer { if accessing { url.stopAccessingSecurityScopedResource() } }
            let data = try Data(contentsOf: url)
            var extraction = try PDFTextExtractor.extract(data: data)
            if extraction.needsOCR {
                // 5.10.2：テキストがほとんど取れない場合はスキャン PDF とみなし、OCR に回す
                let pages = try PDFTextExtractor.renderPages(data: data)
                extraction.text = try await TextRecognizer.recognize(pages: pages)
            }
            extractedType = .pdf
            extracted = IdentifiedText(text: extraction.text)
        } catch {
            self.error = error
        }
    }

    private func handlePhoto(_ item: PhotosPickerItem) async {
        isWorking = true
        defer { isWorking = false }
        do {
            guard let data = try await item.loadTransferable(type: Data.self), let image = Self.cgImage(from: data) else {
                throw PDFTextExtractor.ExtractionError.unreadable
            }
            await handleImage(image)
        } catch {
            self.error = error
        }
    }

    private func handleImage(_ image: CGImage) async {
        do {
            let text = try await TextRecognizer.recognize(pages: [image])
            extractedType = .image
            extracted = IdentifiedText(text: text)
        } catch {
            self.error = error
        }
    }

    static func cgImage(from data: Data) -> CGImage? {
        guard let source = CGImageSourceCreateWithData(data as CFData, nil) else { return nil }
        return CGImageSourceCreateImageAtIndex(source, 0, nil)
    }
}

#if os(iOS)
struct CameraPicker: UIViewControllerRepresentable {
    let onCapture: (CGImage) -> Void
    @Environment(\.dismiss) private var dismiss

    func makeUIViewController(context: Context) -> UIImagePickerController {
        let picker = UIImagePickerController()
        picker.sourceType = .camera
        picker.delegate = context.coordinator
        return picker
    }

    func updateUIViewController(_ controller: UIImagePickerController, context: Context) {}

    func makeCoordinator() -> Coordinator { Coordinator(self) }

    final class Coordinator: NSObject, UIImagePickerControllerDelegate, UINavigationControllerDelegate {
        let parent: CameraPicker
        init(_ parent: CameraPicker) { self.parent = parent }

        func imagePickerController(_ picker: UIImagePickerController, didFinishPickingMediaWithInfo info: [UIImagePickerController.InfoKey: Any]) {
            if let image = (info[.originalImage] as? UIImage)?.cgImage { parent.onCapture(image) }
            parent.dismiss()
        }

        func imagePickerControllerDidCancel(_ picker: UIImagePickerController) {
            parent.dismiss()
        }
    }
}
#endif

/// 抽出テキストの確認・編集 → 保存（`POST /research-materials`）
struct MaterialTextEditorView: View {
    @Environment(AppModel.self) private var model
    @Environment(\.dismiss) private var dismiss
    let companyId: ID
    let type: ResearchMaterialType
    let sourceUrl: String?
    let initialText: String
    let onSaved: () -> Void

    @ViewState private var title = ""
    @ViewState private var text = ""
    @ViewState private var error: Error?

    var body: some View {
        Form {
            TextField("タイトル（例：採用ページ）", text: $title)
            Section {
                TextEditor(text: $text).frame(minHeight: 280)
            } header: {
                HStack {
                    Text("本文（不要な部分を削除してください）")
                    Spacer()
                    Text("\(text.count) 字").monospacedDigit()
                }
            }
            if let message = ResearchLengthAdvice.evaluate(text).message {
                Section { Text(message).font(.footnote).foregroundStyle(.orange) }
            }
            if let error { ErrorBanner(error: error) }
            AsyncButton("保存") {
                do {
                    _ = try await model.api.createResearchMaterial(ResearchMaterialInput(
                        companyId: companyId, title: title, type: type, sourceUrl: sourceUrl, contentRaw: text))
                    onSaved()
                    dismiss()
                } catch {
                    self.error = error
                }
            }
            .disabled(title.trimmingCharacters(in: .whitespaces).isEmpty || text.trimmingCharacters(in: .whitespaces).isEmpty)
        }
        .navigationTitle("本文の確認")
        .onAppear { if text.isEmpty { text = initialText } }
    }
}
