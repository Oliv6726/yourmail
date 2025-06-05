import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";

const geist = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "📬 YourMail - Decentralized Email",
  description: "A modern, decentralized email client built with Next.js",
  viewport: "width=device-width, initial-scale=1, maximum-scale=1",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="h-full">
      <body
        className={`${geist.variable} ${geistMono.variable} antialiased h-full bg-background text-foreground`}
      >
        <div className="h-full min-h-screen">{children}</div>
      </body>
    </html>
  );
}
