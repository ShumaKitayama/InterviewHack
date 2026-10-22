// DesignDocument/5_FrontendDesign.md 5.4.1 ルート構成 / 3_UIUXDesign.md 3.2.1 グローバルナビ（タブバー）
import InterviewHackKit
import SwiftUI

public struct RootView: View {
    @ViewState private var model: AppModel
    @AppStorage("interviewhack.onboardingCompleted") private var onboardingCompleted = false

    public init(model: AppModel) {
        _model = ViewState(initialValue: model)
    }

    public var body: some View {
        Group {
            if model.isRestoring {
                ProgressView("読み込み中")
            } else if model.currentUser == nil {
                NavigationStack { LoginView() }
            } else if !onboardingCompleted {
                NavigationStack { OnboardingView { onboardingCompleted = true } }
            } else {
                MainTabView()
            }
        }
        .environment(model)
        .task { await model.observeAuthState() }
    }
}

/// タブは 5 つ（Home / 対策 / 記録 / バンク / データ）。設定は Home のナビゲーションバーから開く
struct MainTabView: View {
    @Environment(AppModel.self) private var model

    var body: some View {
        VStack(spacing: 0) {
            if !model.connectivity.isOnline {
                OfflineBanner()
            }
            TabView {
                NavigationStack { HomeView() }
                    .tabItem { Label("Home", systemImage: "house") }
                NavigationStack { PrepHubView() }
                    .tabItem { Label("対策", systemImage: "target") }
                NavigationStack { InterviewListView() }
                    .tabItem { Label("記録", systemImage: "square.and.pencil") }
                NavigationStack { QAListView() }
                    .tabItem { Label("バンク", systemImage: "rectangle.stack") }
                NavigationStack { DataHubView() }
                    .tabItem { Label("データ", systemImage: "folder") }
            }
        }
    }
}

/// 「対策」タブ：面接対策ルーム（S-Prep-01）と ES 作成ルーム（S-ES-01）
struct PrepHubView: View {
    var body: some View {
        List {
            Section {
                NavigationLink {
                    PrepRoomView()
                } label: {
                    Label("面接対策ルーム（想定Q&A）", systemImage: "person.2.wave.2")
                }
                NavigationLink {
                    ESProjectListView()
                } label: {
                    Label("ES作成ルーム", systemImage: "doc.text")
                }
            }
        }
        .navigationTitle("対策")
    }
}

/// 「データ」タブ：自己資産DB（S-Data-01）と企業DB（S-Data-03）
struct DataHubView: View {
    var body: some View {
        List {
            NavigationLink {
                AssetListView()
            } label: {
                Label("自己資産", systemImage: "person.text.rectangle")
            }
            NavigationLink {
                CompanyListView()
            } label: {
                Label("企業", systemImage: "building.2")
            }
        }
        .navigationTitle("データ")
    }
}
