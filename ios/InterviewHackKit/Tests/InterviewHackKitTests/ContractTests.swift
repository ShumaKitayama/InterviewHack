// 契約テスト：BFF が実際に返した JSON（backend の TestContractFixtures が書き出したもの）を
// Swift のモデルでデコードできることを確認する。DesignDocument/7_BackendDesign.md 7.5 / 7.7
// fixture の再生成：backend で `CONTRACT_OUT=<このディレクトリ>/Fixtures/contract go test ./internal/app -run TestContractFixtures`
import Foundation
import Testing
@testable import InterviewHackKit

@Suite("BFF との契約（実レスポンスのデコード）")
struct ContractTests {
    static let dir = URL(fileURLWithPath: #filePath).deletingLastPathComponent().appendingPathComponent("Fixtures/contract")

    func decode<T: Decodable>(_ type: T.Type, _ name: String) throws -> T {
        let data = try Data(contentsOf: Self.dir.appendingPathComponent(name + ".json"))
        return try JSONCoding.makeDecoder().decode(T.self, from: data)
    }

    @Test("アカウント・自己資産・企業・資料")
    func resources() throws {
        let me = try decode(Me.self, "me")
        #expect(me.displayName == "テスト")
        let tags = try decode(ItemsResponse<AssetTag>.self, "asset_tags")
        #expect(tags.items.first?.name == "ガクチカ")
        let asset = try decode(Asset.self, "asset")
        #expect(asset.tagIds?.count == 1)
        let assets = try decode(Page<Asset>.self, "assets")
        #expect(assets.items.count == 1 && assets.nextCursor != nil)
        _ = try decode(Company.self, "company")
        let material = try decode(ResearchMaterial.self, "research_material")
        #expect(material.type == .url)
    }

    @Test("ES・版・面接・Q&A")
    func esAndInterviews() throws {
        let project = try decode(ESProject.self, "es_project")
        #expect(project.status == .notStarted || project.status == .draft)
        #expect(project.deadline == CalendarDate(year: 2026, month: 12, day: 1))
        let sections = try decode(Page<ESSection>.self, "es_sections")
        #expect(sections.items.count == 1)
        let section = try decode(ESSection.self, "es_section")
        #expect(section.contentDraft == "本文")
        let version = try decode(ESSectionVersion.self, "es_section_version")
        #expect(version.source == .manual)
        let versions = try decode(ItemsResponse<ESSectionVersion>.self, "es_section_versions")
        #expect(versions.items.count == 1)
        let interview = try decode(Interview.self, "interview")
        #expect(interview.result == .pending)
        _ = try decode(Page<InterviewLog>.self, "interview_logs")
        let qa = try decode(QA.self, "qa")
        #expect(qa.reviewStatus == .new && qa.sourceType == .preGenerated)
    }

    @Test("AI連携：作成・一覧・解析・保存")
    func aiLink() throws {
        let created = try decode(PromptText.self, "prompt_created")
        #expect(created.templateVersion == "qa-v1" && created.warnings != nil)
        _ = try decode(PromptText.self, "prompt_text")
        let sessions = try decode(Page<PromptSessionSummary>.self, "prompt_sessions")
        #expect(sessions.items.first?.status == .pasted)
        #expect(sessions.items.first?.targetLabel?.isEmpty == false)
        let one = try decode(PromptSessionSummary.self, "prompt_session")
        #expect(one.useCase == .qaGenerate)
        let qa = try decode(ParseResponse.self, "parse_qa")
        #expect(qa.parseResult == .success && qa.preview.items?.first?.intentText == "i")
        let review = try decode(ParseResponse.self, "parse_review")
        #expect(review.preview.issues == ["x"] && review.preview.improvedAnswerText == "z")
        let research = try decode(ParseResponse.self, "parse_research")
        #expect(research.preview.highlights == ["a"] && research.preview.idealCandidate == "b")
        let text = try decode(ParseResponse.self, "parse_text")
        #expect(text.preview.text == "自由な文章")
        let commit = try decode(CommitResponse.self, "commit")
        #expect(commit.status == .saved && commit.savedRefs["qa"]?.count == 1)
        let abandon = try decode(SessionStatusResponse.self, "abandon")
        #expect(abandon.status == .abandoned)
    }

    @Test("ダッシュボードと共有リンク")
    func dashboardAndShare() throws {
        let dashboard = try decode(Dashboard.self, "dashboard")
        #expect(dashboard.latestSummary?.content == "- 結論から")
        #expect(dashboard.upcomingInterviews.count == 1)
        let created = try decode(CreatedShareLink.self, "share_created")
        #expect(created.url.contains("/s/"))
        let links = try decode(ItemsResponse<ESShareLink>.self, "share_links")
        #expect(links.items.first?.isActive == true)
    }

    @Test("エラー形式（7.7.1）")
    func errors() throws {
        for (name, code) in [("error_conflict", "SESSION_STATE_CONFLICT"), ("error_validation", "VALIDATION_ERROR"),
                             ("error_bad_request", "VALIDATION_ERROR"), ("error_unauthorized", "UNAUTHORIZED")] {
            let body = try decode(ErrorBody.self, name)
            #expect(body.code == code)
            #expect(body.requestId?.isEmpty == false)
            #expect(APIErrorCode(rawValue: body.code ?? "") != .other(code))
        }
    }
}
