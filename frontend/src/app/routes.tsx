// DesignDocument/5_FrontendDesign.md 5.4 ルーティング設計
import { createBrowserRouter } from "react-router-dom";
import { AuthGuard } from "../components/AuthGuard";
import { AssetFormPage } from "../pages/AssetFormPage";
import { AssetsPage } from "../pages/AssetsPage";
import { AuthCallbackPage } from "../pages/AuthCallbackPage";
import { BillingPage } from "../pages/BillingPage";
import { CompaniesPage } from "../pages/CompaniesPage";
import { CompanyFormPage } from "../pages/CompanyFormPage";
import { ESEditorPage } from "../pages/ESEditorPage";
import { ESNewPage } from "../pages/ESNewPage";
import { ESPage } from "../pages/ESPage";
import { HomePage } from "../pages/HomePage";
import { InterviewDetailPage } from "../pages/InterviewDetailPage";
import { InterviewLogNewPage } from "../pages/InterviewLogNewPage";
import { InterviewNewPage } from "../pages/InterviewNewPage";
import { InterviewsPage } from "../pages/InterviewsPage";
import { LoginPage } from "../pages/LoginPage";
import { PrepPage } from "../pages/PrepPage";
import { PrepResultPage } from "../pages/PrepResultPage";
import { QADetailPage } from "../pages/QADetailPage";
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
      { path: "assets/new", element: <AssetFormPage /> },
      { path: "assets/:id", element: <AssetFormPage /> },

      { path: "companies", element: <CompaniesPage /> },
      { path: "companies/new", element: <CompanyFormPage /> },
      { path: "companies/:id", element: <CompanyFormPage /> },

      { path: "es", element: <ESPage /> },
      { path: "es/new", element: <ESNewPage /> },
      { path: "es/:id", element: <ESEditorPage /> },

      { path: "prep", element: <PrepPage /> },
      { path: "prep/result/:generationId", element: <PrepResultPage /> },

      { path: "interviews", element: <InterviewsPage /> },
      { path: "interviews/new", element: <InterviewNewPage /> },
      { path: "interviews/:id", element: <InterviewDetailPage /> },
      { path: "interviews/:id/logs/new", element: <InterviewLogNewPage /> },

      { path: "qa", element: <QAPage /> },
      { path: "qa/:id", element: <QADetailPage /> },

      { path: "review", element: <ReviewPage /> },
      { path: "billing", element: <BillingPage /> },
      { path: "settings", element: <SettingsPage /> },
    ],
  },
]);
