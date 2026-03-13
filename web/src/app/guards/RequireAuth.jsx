import { Navigate, useLocation } from "react-router-dom";

import { hasToken } from "../../services/auth";

export function RequireAuth({ children }) {
  const location = useLocation();

  if (!hasToken()) {
    return <Navigate to="/login" replace state={{ from: location }} />;
  }

  return children;
}
