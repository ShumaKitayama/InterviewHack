// DesignDocument/5_FrontendDesign.md 5.5 認証設計（Firebase Auth）
// Firebase の実装はアプリ本体（ios/App の FirebaseAuthService）に置き、ここでは抽象だけを定義する。
import Foundation

/// API 呼び出しのたびに ID トークンを返す（5.5.2）。未ログインなら nil。
public protocol AuthTokenProvider: Sendable {
    func idToken(forceRefresh: Bool) async throws -> String?
}

public struct AuthUser: Equatable, Sendable {
    public var uid: String
    public var email: String?

    public init(uid: String, email: String?) {
        self.uid = uid
        self.email = email
    }
}

public enum AuthServiceError: Error, Equatable, Sendable, LocalizedError {
    case invalidCredentials
    case emailAlreadyInUse
    case weakPassword
    case requiresRecentLogin
    case network
    case unknown(String)

    public var errorDescription: String? {
        switch self {
        case .invalidCredentials: return "メールアドレスまたはパスワードが正しくありません"
        case .emailAlreadyInUse: return "このメールアドレスはすでに登録されています"
        case .weakPassword: return "パスワードは 6 文字以上にしてください"
        case .requiresRecentLogin: return "安全のため、もう一度ログインしてから操作してください"
        case .network: return "通信できませんでした"
        case .unknown(let message): return message
        }
    }
}

/// メールアドレス＋パスワードの認証（D5）。Google / Sign in with Apple は将来追加する。
public protocol AuthService: AuthTokenProvider {
    var currentUser: AuthUser? { get }
    /// ログイン状態の変化（`addStateDidChangeListener` 相当）
    func authStateChanges() -> AsyncStream<AuthUser?>
    func signUp(email: String, password: String) async throws -> AuthUser
    func signIn(email: String, password: String) async throws -> AuthUser
    func sendPasswordReset(email: String) async throws
    func signOut() throws
    /// 5.5.4：`DELETE /me` の成功後に Firebase のユーザーを削除する
    func deleteCurrentUser() async throws
}

/// テストとプレビュー用のメモリ上の実装
public final class InMemoryAuthService: AuthService, @unchecked Sendable {
    private let lock = NSLock()
    private var accounts: [String: (password: String, uid: String)] = [:]
    private var user: AuthUser?
    private var continuations: [UUID: AsyncStream<AuthUser?>.Continuation] = [:]
    private var tokenCounter = 0

    public private(set) var forcedRefreshCount = 0

    public init(signedInAs user: AuthUser? = nil) {
        self.user = user
    }

    public var currentUser: AuthUser? {
        lock.withLock { user }
    }

    public func authStateChanges() -> AsyncStream<AuthUser?> {
        AsyncStream { continuation in
            let id = UUID()
            lock.withLock {
                continuations[id] = continuation
                continuation.yield(user)
            }
            continuation.onTermination = { [weak self] _ in
                self?.lock.withLock { _ = self?.continuations.removeValue(forKey: id) }
            }
        }
    }

    public func idToken(forceRefresh: Bool) async throws -> String? {
        lock.withLock {
            guard let user else { return nil }
            if forceRefresh {
                forcedRefreshCount += 1
                tokenCounter += 1
            }
            return "token-\(user.uid)-\(tokenCounter)"
        }
    }

    public func signUp(email: String, password: String) async throws -> AuthUser {
        guard password.count >= 6 else { throw AuthServiceError.weakPassword }
        let newUser: AuthUser = try lock.withLock {
            if accounts[email] != nil { throw AuthServiceError.emailAlreadyInUse }
            let uid = "uid-\(accounts.count + 1)"
            accounts[email] = (password, uid)
            return AuthUser(uid: uid, email: email)
        }
        setUser(newUser)
        return newUser
    }

    public func signIn(email: String, password: String) async throws -> AuthUser {
        let signedIn: AuthUser = try lock.withLock {
            guard let account = accounts[email], account.password == password else {
                throw AuthServiceError.invalidCredentials
            }
            return AuthUser(uid: account.uid, email: email)
        }
        setUser(signedIn)
        return signedIn
    }

    public func sendPasswordReset(email: String) async throws {}

    public func signOut() throws {
        setUser(nil)
    }

    public func deleteCurrentUser() async throws {
        lock.withLock {
            if let email = user?.email { accounts[email] = nil }
        }
        setUser(nil)
    }

    private func setUser(_ newValue: AuthUser?) {
        let targets: [AsyncStream<AuthUser?>.Continuation] = lock.withLock {
            user = newValue
            return Array(continuations.values)
        }
        targets.forEach { $0.yield(newValue) }
    }
}
