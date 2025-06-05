"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Loader2, Mail, User, Lock, Server } from "lucide-react";
import { YourMailAPI, type User as ApiUser } from "@/lib/api";

interface AuthFormProps {
  onAuthSuccess: (user: ApiUser, token: string) => void;
}

type AuthMode = "login" | "register";

export function AuthForm({ onAuthSuccess }: AuthFormProps) {
  const [mode, setMode] = useState<AuthMode>("login");
  const [formData, setFormData] = useState({
    username: "",
    email: "",
    password: "",
    confirmPassword: "",
    serverHost: "localhost",
    serverPort: "8080",
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");
  const [api, setApi] = useState(
    () => new YourMailAPI(formData.serverHost, parseInt(formData.serverPort))
  );

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));

    // Update API connection if server details change
    if (name === "serverHost" || name === "serverPort") {
      const host = name === "serverHost" ? value : formData.serverHost;
      const port =
        name === "serverPort"
          ? parseInt(value) || 8080
          : parseInt(formData.serverPort);
      setApi(new YourMailAPI(host, port));
    }

    // Clear error when user starts typing
    if (error) setError("");
  };

  const validateForm = () => {
    if (!formData.username || formData.username.length < 3) {
      return "Username must be at least 3 characters long";
    }
    if (mode === "register") {
      if (!formData.email || !formData.email.includes("@")) {
        return "Please enter a valid email address";
      }
      if (!formData.password || formData.password.length < 6) {
        return "Password must be at least 6 characters long";
      }
      if (formData.password !== formData.confirmPassword) {
        return "Passwords do not match";
      }
    } else {
      if (!formData.password) {
        return "Password is required";
      }
    }
    return null;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const validationError = validateForm();
    if (validationError) {
      setError(validationError);
      return;
    }

    setIsLoading(true);
    setError("");

    try {
      let response;
      if (mode === "register") {
        response = await api.register({
          username: formData.username,
          email: formData.email,
          password: formData.password,
        });
      } else {
        response = await api.login(formData.username, formData.password);
      }

      if (response.success && response.user && response.token) {
        onAuthSuccess(response.user, response.token);
      } else {
        setError(
          response.message ||
            `${mode === "register" ? "Registration" : "Login"} failed`
        );
      }
    } catch (err) {
      console.error(`${mode} error:`, err);
      setError(
        err instanceof Error
          ? err.message
          : `${mode === "register" ? "Registration" : "Login"} failed`
      );
    } finally {
      setIsLoading(false);
    }
  };

  const fillDemoUser = (user: "alice" | "bob") => {
    if (user === "alice") {
      setFormData((prev) => ({
        ...prev,
        username: "alice",
        password: "password123",
        email: "alice@yourmail.local",
      }));
    } else {
      setFormData((prev) => ({
        ...prev,
        username: "bob",
        password: "password456",
        email: "bob@yourmail.local",
      }));
    }
    setMode("login");
  };

  const toggleMode = () => {
    setMode(mode === "login" ? "register" : "login");
    setError("");
    if (mode === "register") {
      setFormData((prev) => ({ ...prev, email: "", confirmPassword: "" }));
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4 bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-gray-900 dark:to-gray-800">
      <Card className="w-full max-w-md shadow-xl">
        <CardHeader className="space-y-1">
          <div className="flex items-center justify-center mb-4">
            <Mail className="h-8 w-8 text-blue-600" />
          </div>
          <CardTitle className="text-2xl text-center">
            {mode === "login" ? "Welcome Back" : "Create Account"}
          </CardTitle>
          <CardDescription className="text-center">
            {mode === "login"
              ? "Sign in to your YourMail account"
              : "Enter your details to create a new account"}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="username">Username</Label>
              <div className="relative">
                <User className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-4 w-4" />
                <Input
                  id="username"
                  name="username"
                  type="text"
                  placeholder="Enter your username"
                  value={formData.username}
                  onChange={handleInputChange}
                  disabled={isLoading}
                  className="pl-10"
                  required
                />
              </div>
            </div>

            {mode === "register" && (
              <div className="space-y-2">
                <Label htmlFor="email">Email</Label>
                <div className="relative">
                  <Mail className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-4 w-4" />
                  <Input
                    id="email"
                    name="email"
                    type="email"
                    placeholder="Enter your email"
                    value={formData.email}
                    onChange={handleInputChange}
                    disabled={isLoading}
                    className="pl-10"
                    required
                  />
                </div>
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-4 w-4" />
                <Input
                  id="password"
                  name="password"
                  type="password"
                  placeholder="Enter your password"
                  value={formData.password}
                  onChange={handleInputChange}
                  disabled={isLoading}
                  className="pl-10"
                  required
                />
              </div>
            </div>

            {mode === "register" && (
              <div className="space-y-2">
                <Label htmlFor="confirmPassword">Confirm Password</Label>
                <div className="relative">
                  <Lock className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-4 w-4" />
                  <Input
                    id="confirmPassword"
                    name="confirmPassword"
                    type="password"
                    placeholder="Confirm your password"
                    value={formData.confirmPassword}
                    onChange={handleInputChange}
                    disabled={isLoading}
                    className="pl-10"
                    required
                  />
                </div>
              </div>
            )}

            {/* Demo User Buttons - only show for login */}
            {mode === "login" && (
              <div className="flex gap-2">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => fillDemoUser("alice")}
                  disabled={isLoading}
                  className="flex-1"
                >
                  Demo: Alice
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => fillDemoUser("bob")}
                  disabled={isLoading}
                  className="flex-1"
                >
                  Demo: Bob
                </Button>
              </div>
            )}

            {/* Server Connection */}
            <div className="border-t pt-4">
              <div className="flex items-center gap-2 mb-2">
                <Server className="h-4 w-4 text-gray-400" />
                <Label className="text-sm font-medium">Server Connection</Label>
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div className="space-y-1">
                  <Label htmlFor="serverHost" className="text-xs">
                    Host
                  </Label>
                  <Input
                    id="serverHost"
                    name="serverHost"
                    type="text"
                    placeholder="localhost"
                    value={formData.serverHost}
                    onChange={handleInputChange}
                    disabled={isLoading}
                    className="text-sm"
                  />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="serverPort" className="text-xs">
                    Port
                  </Label>
                  <Input
                    id="serverPort"
                    name="serverPort"
                    type="number"
                    placeholder="8080"
                    value={formData.serverPort}
                    onChange={handleInputChange}
                    disabled={isLoading}
                    className="text-sm"
                  />
                </div>
              </div>
            </div>

            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            <Button type="submit" className="w-full" disabled={isLoading}>
              {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {mode === "login" ? "Sign In" : "Create Account"}
            </Button>

            <div className="text-center">
              <Button
                type="button"
                variant="link"
                onClick={toggleMode}
                disabled={isLoading}
                className="text-sm"
              >
                {mode === "login"
                  ? "Don't have an account? Sign up"
                  : "Already have an account? Sign in"}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
