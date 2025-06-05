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
import { ServerConnection } from "@/types/mail";

interface LoginFormProps {
  onLogin: (connection: ServerConnection, password: string) => void;
  isLoading?: boolean;
}

export function LoginForm({ onLogin, isLoading = false }: LoginFormProps) {
  const [formData, setFormData] = useState({
    username: "",
    password: "",
    serverHost: "localhost",
    serverPort: "8080",
  });

  const [errors, setErrors] = useState<Record<string, string>>({});

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrors({});

    // Basic validation
    const newErrors: Record<string, string> = {};
    if (!formData.username.trim()) newErrors.username = "Username is required";
    if (!formData.password.trim()) newErrors.password = "Password is required";
    if (!formData.serverHost.trim())
      newErrors.serverHost = "Server host is required";
    if (!formData.serverPort.trim())
      newErrors.serverPort = "Server port is required";

    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors);
      return;
    }

    // Create connection object and pass password
    const connection: ServerConnection = {
      isConnected: true,
      host: formData.serverHost,
      port: parseInt(formData.serverPort),
      user: {
        username: formData.username,
        serverHost: formData.serverHost,
      },
    };

    // Pass both connection and password to the login handler
    onLogin(connection, formData.password);
  };

  const handleInputChange = (field: string, value: string) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
    if (errors[field]) {
      setErrors((prev) => ({ ...prev, [field]: "" }));
    }
  };

  const fillDemoUser = (user: "alice" | "bob") => {
    if (user === "alice") {
      setFormData((prev) => ({
        ...prev,
        username: "alice",
        password: "password123",
      }));
    } else {
      setFormData((prev) => ({
        ...prev,
        username: "bob",
        password: "password456",
      }));
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4 bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-gray-900 dark:to-gray-800">
      <Card className="w-full max-w-md shadow-xl">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl font-bold text-primary">
            📬 YourMail
          </CardTitle>
          <CardDescription>
            Connect to your decentralized mail server
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                type="text"
                placeholder="Enter username (alice or bob)"
                value={formData.username}
                onChange={(e) => handleInputChange("username", e.target.value)}
                className={errors.username ? "border-red-500" : ""}
                disabled={isLoading}
              />
              {errors.username && (
                <p className="text-sm text-red-500">{errors.username}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                placeholder="Enter password"
                value={formData.password}
                onChange={(e) => handleInputChange("password", e.target.value)}
                className={errors.password ? "border-red-500" : ""}
                disabled={isLoading}
              />
              {errors.password && (
                <p className="text-sm text-red-500">{errors.password}</p>
              )}
            </div>

            {/* Demo User Buttons */}
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

            <div className="grid grid-cols-2 gap-2">
              <div className="space-y-2">
                <Label htmlFor="serverHost">Server Host</Label>
                <Input
                  id="serverHost"
                  type="text"
                  placeholder="localhost"
                  value={formData.serverHost}
                  onChange={(e) =>
                    handleInputChange("serverHost", e.target.value)
                  }
                  className={errors.serverHost ? "border-red-500" : ""}
                  disabled={isLoading}
                />
                {errors.serverHost && (
                  <p className="text-sm text-red-500">{errors.serverHost}</p>
                )}
              </div>

              <div className="space-y-2">
                <Label htmlFor="serverPort">Port</Label>
                <Input
                  id="serverPort"
                  type="number"
                  placeholder="8080"
                  value={formData.serverPort}
                  onChange={(e) =>
                    handleInputChange("serverPort", e.target.value)
                  }
                  className={errors.serverPort ? "border-red-500" : ""}
                  disabled={isLoading}
                />
                {errors.serverPort && (
                  <p className="text-sm text-red-500">{errors.serverPort}</p>
                )}
              </div>
            </div>

            <Button type="submit" className="w-full" disabled={isLoading}>
              {isLoading ? "Connecting..." : "Connect to Server"}
            </Button>
          </form>

          <div className="mt-6 text-center text-sm text-muted-foreground">
            <p>
              Don&apos;t have a server? <br />
              <span className="font-mono text-xs bg-muted px-2 py-1 rounded">
                go run cmd/server/main.go
              </span>
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
