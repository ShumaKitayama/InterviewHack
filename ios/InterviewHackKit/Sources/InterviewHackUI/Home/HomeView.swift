// DesignDocument/3_UIUXDesign.md 3.5.1 Home（S-Home-01）/ 7_BackendDesign.md 7.5.4 Dashboard
import InterviewHackKit
import SwiftUI

struct HomeView: View {
    @Environment(AppModel.self) private var model
    @ViewState private var dashboard: CachedResource<Dashboard>?
    @ViewState private var drafts: [DraftRecord] = []
    @ViewState private var summaryFlow: AILinkFlow?
    @ViewState private var resumeFlow: AILinkFlow?

    var body: some View {
        List {
            if let resource = dashboard {
                if let error = resource.error, resource.value == nil {
                    ErrorBanner(error: error) { Task { await resource.refresh() } }
                }
                let value = resource.value ?? Dashboard()
                Section {
                    RefreshingIndicator(isRefreshing: resource.isRefreshing)
                    NavigationLink { PrepRoomView() } label: { Label("対策を始める", systemImage: "target") }
                    NavigationLink { InterviewListView() } label: { Label("振り返る", systemImage: "square.and.pencil") }
                    NavigationLink { QAReviewView(filter: QAFilter(reviewStatus: .new)) } label: {
                        Label("復習する", systemImage: "rectangle.stack")
                    }
                }
                if value.pendingPromptSessions.count > 0 {
                    Section("未完了の AI連携（\(value.pendingPromptSessions.count) 件）") {
                        if let latest = value.pendingPromptSessions.items.first {
                            Button {
                                let flow = model.makeFlow(latest.useCase)
                                resumeFlow = flow
                                Task { await flow.resume(latest) }
                            } label: {
                                VStack(alignment: .leading) {
                                    Text(latest.useCase.label).font(.headline)
                                    if let target = latest.targetLabel { Text(target).font(.subheadline) }
                                }
                            }
                        }
                        NavigationLink("すべて見る") { PendingSessionsView() }
                    }
                }
                if !drafts.isEmpty {
                    Section("送信していない下書き") {
                        Text("端末に \(drafts.count) 件の下書きがあります。各画面から確認して送信してください")
                            .font(.callout)
                    }
                }
                Section("次の面接") {
                    if value.upcomingInterviews.isEmpty {
                        Text("予定されている面接はありません").foregroundStyle(.secondary)
                    }
                    ForEach(value.upcomingInterviews) { interview in
                        NavigationLink {
                            InterviewDetailView(interview: interview)
                        } label: {
                            VStack(alignment: .leading) {
                                Text(interview.interviewDate.description).font(.headline)
                                if let phase = interview.phase { Text(phase).font(.subheadline) }
                            }
                        }
                    }
                }
                Section("次回直すべきポイント") {
                    if let summary = value.latestSummary {
                        Text(summary.content).font(.callout)
                        if let createdAt = summary.createdAt {
                            Text(createdAt.formatted(date: .abbreviated, time: .omitted)).font(.caption).foregroundStyle(.secondary)
                        }
                    } else {
                        Text("面接ログを添削すると、次回のポイントを要約できます").foregroundStyle(.secondary)
                    }
                    Button("要約を更新する（AI連携）") {
                        let flow = model.makeFlow(.summaryGenerate)
                        summaryFlow = flow
                        Task { await flow.start(PromptSessionRequest(useCase: .summaryGenerate)) }
                    }
                    .disabled(!model.connectivity.isOnline)
                }
                if !value.recentQa.isEmpty {
                    Section("最近の Q&A") {
                        ForEach(value.recentQa) { qa in
                            NavigationLink(qa.questionText) { QADetailView(qa: qa) }
                        }
                    }
                }
            }
        }
        .navigationTitle("Home")
        .toolbar {
            ToolbarItem(placement: ToolbarPlacement.trailing) {
                NavigationLink {
                    SettingsView()
                } label: {
                    Image(systemName: "gearshape").accessibilityLabel("設定")
                }
            }
        }
        .task {
            if dashboard == nil {
                let api = model.api
                dashboard = model.resource(.dashboard) { try await api.dashboard() }
            }
            drafts = (try? model.drafts.pendingDrafts()) ?? []
            await dashboard?.refresh()
        }
        .refreshable { await dashboard?.refresh() }
        .aiLinkSheet($summaryFlow, title: UseCase.summaryGenerate.label) { _ in Task { await dashboard?.refresh() } }
        .aiLinkSheet($resumeFlow, title: "AI連携の再開") { _ in Task { await dashboard?.refresh() } }
    }
}
