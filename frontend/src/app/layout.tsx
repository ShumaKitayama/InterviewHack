// DesignDocument/5_FrontendDesign.md 5.10, 3_UIUXDesign.md 3.2.1 - グローバルナビ（下部タブ）
import { NavLink, Outlet } from "react-router-dom";

const navItems = [
  {
    to: "/",
    label: "ホーム",
    icon: (
      <svg
        className="w-6 h-6"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth={1.8}
      >
        <path d="M3 9.5L12 3l9 6.5V20a1 1 0 01-1 1h-5v-6H9v6H4a1 1 0 01-1-1V9.5z" />
      </svg>
    ),
    iconActive: (
      <svg className="w-6 h-6" viewBox="0 0 24 24" fill="currentColor">
        <path d="M3 9.5L12 3l9 6.5V20a1 1 0 01-1 1h-5v-6H9v6H4a1 1 0 01-1-1V9.5z" />
      </svg>
    ),
  },
  {
    to: "/assets",
    label: "自己資産",
    icon: (
      <svg
        className="w-6 h-6"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth={1.8}
      >
        <path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2" />
        <circle cx="12" cy="7" r="4" />
      </svg>
    ),
    iconActive: (
      <svg className="w-6 h-6" viewBox="0 0 24 24" fill="currentColor">
        <path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2h16z" />
        <circle cx="12" cy="7" r="4" />
      </svg>
    ),
  },
  {
    to: "/companies",
    label: "企業DB",
    icon: (
      <svg
        className="w-6 h-6"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth={1.8}
      >
        <path d="M4 21V4a1 1 0 011-1h9a1 1 0 011 1v17" />
        <path d="M15 21V10a1 1 0 011-1h3a1 1 0 011 1v11" />
        <path d="M2 21h20" />
        <path d="M7 7h3M7 11h3M7 15h3" />
      </svg>
    ),
    iconActive: (
      <svg className="w-6 h-6" viewBox="0 0 24 24" fill="currentColor">
        <path d="M4 21V4a1 1 0 011-1h9a1 1 0 011 1v17H4z" opacity="0.9" />
        <path d="M15 21V10a1 1 0 011-1h3a1 1 0 011 1v11H15z" opacity="0.7" />
        <path
          d="M2 21h20"
          stroke="currentColor"
          strokeWidth={1.8}
          fill="none"
        />
        <path
          d="M7 7h3M7 11h3M7 15h3"
          stroke="white"
          strokeWidth={1.5}
          fill="none"
        />
      </svg>
    ),
  },
  {
    to: "/qa",
    label: "Q&A",
    icon: (
      <svg
        className="w-6 h-6"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth={1.8}
      >
        <path d="M4 19.5A2.5 2.5 0 016.5 17H20" />
        <path d="M6.5 2H20v20H6.5A2.5 2.5 0 014 19.5v-15A2.5 2.5 0 016.5 2z" />
        <path d="M8 7h8M8 11h6" />
      </svg>
    ),
    iconActive: (
      <svg className="w-6 h-6" viewBox="0 0 24 24" fill="currentColor">
        <path d="M4 19.5A2.5 2.5 0 016.5 17H20V2H6.5A2.5 2.5 0 014 4.5v15z" />
        <path
          d="M4 19.5A2.5 2.5 0 016.5 17H20v5H6.5A2.5 2.5 0 014 19.5z"
          opacity="0.6"
        />
        <path d="M8 7h8M8 11h6" stroke="white" strokeWidth={1.5} fill="none" />
      </svg>
    ),
  },
];

export function Layout() {
  return (
    <div className="min-h-screen bg-gray-50 pb-20">
      <main className="max-w-lg mx-auto">
        <Outlet />
      </main>
      <nav className="fixed bottom-0 left-0 right-0 bg-white border-t border-gray-100 safe-area-bottom z-40">
        <div className="max-w-lg mx-auto flex justify-around">
          {navItems.map(({ to, label, icon, iconActive }) => (
            <NavLink
              key={to}
              to={to}
              end={to === "/"}
              className={({ isActive }) =>
                `flex flex-col items-center py-2 px-3 min-w-[64px] text-xs font-medium transition-colors ${
                  isActive ? "text-orange-500" : "text-gray-400"
                }`
              }
            >
              {({ isActive }) => (
                <>
                  <span className="mb-0.5">{isActive ? iconActive : icon}</span>
                  <span>{label}</span>
                </>
              )}
            </NavLink>
          ))}
        </div>
      </nav>
    </div>
  );
}
