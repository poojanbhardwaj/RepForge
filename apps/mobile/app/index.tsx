import { ProfileScreen } from "../src/profile/ProfileScreen";
import { AuthenticatedApp } from "../src/auth/AuthenticatedApp";
import { mobileAuthMode } from "../src/auth/config";

export default function IndexScreen() {
  return mobileAuthMode === "oidc" ? <AuthenticatedApp /> : <ProfileScreen />;
}
