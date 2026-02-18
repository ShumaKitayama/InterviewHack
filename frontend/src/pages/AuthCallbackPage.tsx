// DesignDocument/5_FrontendDesign.md 5.5.1 /auth/callback
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";

export function AuthCallbackPage() {
  const navigate = useNavigate();

  useEffect(() => {
    const timer = setTimeout(() => navigate("/", { replace: true }), 1000);
    return () => clearTimeout(timer);
  }, [navigate]);

  return (
    <div className="flex items-center justify-center min-h-screen">
      <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
    </div>
  );
}
