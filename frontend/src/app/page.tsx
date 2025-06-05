"use client";

import { useState, useEffect } from "react";
import { AuthForm } from "@/components/auth/AuthForm";
import { MailClient } from "@/components/mail/MailClient";
import { YourMailAPI, type User } from "@/lib/api";

export default function HomePage() {
  const [user, setUser] = useState<User | null>(null);
  const [api, setApi] = useState<YourMailAPI | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Check for existing authentication on load
  useEffect(() => {
    const checkAuth = async () => {
      try {
        const apiInstance = new YourMailAPI();
        if (apiInstance.isAuthenticated()) {
          // Try to get user profile to verify token is still valid
          const userProfile = await apiInstance.getProfile();
          setUser(userProfile);
          setApi(apiInstance);
        }
      } catch (error) {
        console.error("Auth check failed:", error);
        // Clear invalid token
        const apiInstance = new YourMailAPI();
        apiInstance.clearToken();
      } finally {
        setIsLoading(false);
      }
    };

    checkAuth();
  }, []);

  const handleAuthSuccess = async (userData: User, token: string) => {
    try {
      setIsLoading(true);

      // Create API instance and set token
      const apiInstance = new YourMailAPI();
      apiInstance.setToken(token);

      // Verify token works by getting profile
      const userProfile = await apiInstance.getProfile();

      setUser(userProfile);
      setApi(apiInstance);
    } catch (error) {
      console.error("Auth verification failed:", error);
      // Clear invalid token
      const apiInstance = new YourMailAPI();
      apiInstance.clearToken();
      alert("Authentication failed. Please try again.");
    } finally {
      setIsLoading(false);
    }
  };

  const handleLogout = async () => {
    if (api) {
      await api.logout();
    }
    setUser(null);
    setApi(null);
  };

  // Show loading spinner while checking authentication
  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-gray-600">Checking authentication...</p>
        </div>
      </div>
    );
  }

  // Show auth form if not authenticated
  if (!user || !api) {
    return <AuthForm onAuthSuccess={handleAuthSuccess} />;
  }

  // Show mail client if authenticated
  return <MailClient user={user} api={api} onLogout={handleLogout} />;
}
