// DesignDocument/5_FrontendDesign.md 5.5 認証設計（Firebase Auth：メールアドレス＋パスワード）
// トークンとセッションの保存は Firebase SDK に任せる（Keychain。5.5.3）
import FirebaseAuth
import Foundation
import InterviewHackKit

final class FirebaseAuthService: AuthService, @unchecked Sendable {
    private var auth: Auth { Auth.auth() }

    var currentUser: AuthUser? {
        auth.currentUser.map(Self.makeUser)
    }

    func authStateChanges() -> AsyncStream<AuthUser?> {
        AsyncStream { continuation in
            let handle = auth.addStateDidChangeListener { _, user in
                continuation.yield(user.map(Self.makeUser))
            }
            continuation.onTermination = { _ in
                Auth.auth().removeStateDidChangeListener(handle)
            }
        }
    }

    /// 5.5.2：API を呼ぶたびに ID トークンを取得する（期限が近いときは SDK が自動で更新する）
    func idToken(forceRefresh: Bool) async throws -> String? {
        guard let user = auth.currentUser else { return nil }
        do {
            return try await user.getIDTokenResult(forcingRefresh: forceRefresh).token
        } catch {
            throw Self.map(error)
        }
    }

    func signUp(email: String, password: String) async throws -> AuthUser {
        do {
            return Self.makeUser(try await auth.createUser(withEmail: email, password: password).user)
        } catch {
            throw Self.map(error)
        }
    }

    func signIn(email: String, password: String) async throws -> AuthUser {
        do {
            return Self.makeUser(try await auth.signIn(withEmail: email, password: password).user)
        } catch {
            throw Self.map(error)
        }
    }

    func sendPasswordReset(email: String) async throws {
        do {
            try await auth.sendPasswordReset(withEmail: email)
        } catch {
            throw Self.map(error)
        }
    }

    func signOut() throws {
        do {
            try auth.signOut()
        } catch {
            throw Self.map(error)
        }
    }

    /// 5.5.4：`DELETE /me` の成功後に呼ぶ。直近のログインが必要な場合は requiresRecentLogin
    func deleteCurrentUser() async throws {
        guard let user = auth.currentUser else { return }
        do {
            try await user.delete()
        } catch {
            throw Self.map(error)
        }
    }

    private static func makeUser(_ user: User) -> AuthUser {
        AuthUser(uid: user.uid, email: user.email)
    }

    private static func map(_ error: Error) -> AuthServiceError {
        let nsError = error as NSError
        guard nsError.domain == AuthErrorDomain, let code = AuthErrorCode(rawValue: nsError.code) else {
            return .unknown(nsError.localizedDescription)
        }
        switch code {
        case .wrongPassword, .invalidCredential, .userNotFound, .invalidEmail:
            return .invalidCredentials
        case .emailAlreadyInUse:
            return .emailAlreadyInUse
        case .weakPassword:
            return .weakPassword
        case .requiresRecentLogin:
            return .requiresRecentLogin
        case .networkError:
            return .network
        default:
            return .unknown(nsError.localizedDescription)
        }
    }
}
