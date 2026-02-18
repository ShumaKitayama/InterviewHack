// DesignDocument/5_FrontendDesign.md 5.4 ルーティング設計
import { createBrowserRouter } from "react-router-dom";
import { AuthGuard } from "../components/AuthGuard";
import { AssetsPage } from "../pages/AssetsPage";
import { AuthCallbackPage } from "../pages/AuthCallbackPage";
import { BillingPage } from "../pages/BillingPage";
import { CompaniesPage } from "../pages/CompaniesPage";
import { ESPage } from "../pages/ESPage";
import { HomePage } from "../pages/HomePage";
import { InterviewsPage } from "../pages/InterviewsPage";
import { LoginPage } from "../pages/LoginPage";
import { PrepPage } from "../pages/PrepPage";
import { QAPage } from "../pages/QAPage";
import { ResetPasswordPage } from "../pages/ResetPasswordPage";
import { ReviewPage } from "../pages/ReviewPage";
import { SettingsPage } from "../pages/SettingsPage";
import { Layout } from "./layout";

export const router = createBrowserRouter([
  { path: "/login", element: <LoginPage /> },
  { path: "/auth/callback", element: <AuthCallbackPage /> },
  { path: "/reset-password", element: <ResetPasswordPage /> },
  {
    element: (
      <AuthGuard>
        <Layout />
      </AuthGuard>
    ),
    children: [
      { index: true, element: <HomePage /> },
      { path: "assets", element: <AssetsPage /> },
      { path: "companies", element: <CompaniesPage /> },
      { path: "es", element: <ESPage /> },
      { path: "prep", element: <PrepPage /> },
      { path: "interviews", element: <InterviewsPage /> },
      { path: "qa", element: <QAPage /> },
      { path: "review", element: <ReviewPage /> },
      { path: "billing", element: <BillingPage /> },
      { path: "settings", element: <SettingsPage /> },
    ],
  },
]);
