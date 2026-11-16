import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { Providers }     from "@/lib/providers";
import { PanelProvider } from "@/lib/panel-context";
import { Sidebar }       from "@/components/sidebar";
import { PanelStack }    from "@/components/panels/panel-stack";

const sans = Geist({
  subsets:  ["latin"],
  variable: "--font-sans",
  weight:   ["300", "400", "500", "600", "700"],
});

const mono = Geist_Mono({
  subsets:  ["latin"],
  variable: "--font-mono",
  weight:   ["400", "500"],
});

export const metadata: Metadata = {
  title: "Checkmarx AI Governance",
  description: "AI coding agent governance and audit platform",
  icons: { icon: "/checkmarx-logo.webp" },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className={`${sans.variable} ${mono.variable} min-h-screen antialiased font-sans`} style={{ background: "var(--background)", color: "var(--foreground)" }}>
        <Providers>
          <PanelProvider>
            <div className="flex h-screen overflow-hidden">
              <Sidebar />
              <main className="flex-1 overflow-y-auto p-6 relative">
                {children}
                <PanelStack />
              </main>
            </div>
          </PanelProvider>
        </Providers>
      </body>
    </html>
  );
}
