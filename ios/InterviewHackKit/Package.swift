// swift-tools-version:5.10
// DesignDocument/5_FrontendDesign.md 5.3: Xcode なしで検証できるロジック（InterviewHackKit）と
// SwiftUI の画面（InterviewHackUI）を Swift Package に置き、アプリ本体（ios/App）から利用する。
import PackageDescription

let package = Package(
    name: "InterviewHackKit",
    defaultLocalization: "ja",
    platforms: [.iOS(.v17), .macOS(.v14)],
    products: [
        .library(name: "InterviewHackKit", targets: ["InterviewHackKit"]),
        .library(name: "InterviewHackUI", targets: ["InterviewHackUI"]),
    ],
    targets: [
        .target(name: "InterviewHackKit"),
        .target(name: "InterviewHackUI", dependencies: ["InterviewHackKit"]),
        // Fixtures/contract は BFF の実レスポンス（契約テスト用）。ファイルパスで直接読むのでリソースにしない
        .testTarget(name: "InterviewHackKitTests", dependencies: ["InterviewHackKit"], exclude: ["Fixtures"]),
    ]
)
