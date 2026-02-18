// DesignDocument/5_FrontendDesign.md 5.10 グローバルナビ（下部タブ）
import { NavLink, Outlet } from "react-router-dom";

const navItems = [
  { to: "/", label: "ホーム", icon: "🏠" },
  { to: "/prep", label: "対策", icon: "📝" },
  { to: "/interviews", label: "記録", icon: "🎤" },
  { to: "/qa", label: "バンク", icon: "💡" },
  { to: "/assets", label: "データ", icon: "📊" },
];

export function Layout() {
  return (
    <div className="min-h-screen bg-gray-50 pb-16">
      <main className="max-w-lg mx-auto">
        <Outlet />
      </main>
      <nav className="fixed bottom-0 left-0 right-0 bg-white border-t border-gray-200 safe-area-bottom">
        <div className="max-w-lg mx-auto flex justify-around">
          {navItems.map(({ to, label, icon }) => (
            <NavLink
              key={to}
              to={to}
              end={to === "/"}
              className={({ isActive }) =>
                `flex flex-col items-center py-2 px-3 text-xs transition-colors ${
                  isActive ? "text-blue-600" : "text-gray-500"
                }`
              }
            >
              <span className="text-xl">{icon}</span>
              <span>{label}</span>
            </NavLink>
          ))}
        </div>
      </nav>
    </div>
  );
}
