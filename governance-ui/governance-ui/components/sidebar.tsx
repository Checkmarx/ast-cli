"use client";
import Link from "next/link";
import Image from "next/image";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard, Shield, MonitorPlay, AlertTriangle,
  Puzzle, Users, Bot, Wand2, ClipboardCheck, Coins,
} from "lucide-react";

const NAV = [
  { href: "/",            icon: LayoutDashboard, label: "Overview"    },
  { href: "/policies",    icon: Shield,          label: "Policies"    },
  { href: "/sessions",    icon: MonitorPlay,     label: "Sessions"    },
  { href: "/usage",       icon: Coins,           label: "Token Usage" },
  { href: "/violations",  icon: AlertTriangle,   label: "Violations"  },
  { href: "/compliance",  icon: ClipboardCheck,  label: "Compliance"  },
  { href: "/mcp-servers", icon: Puzzle,          label: "MCP Servers" },
  { href: "/skills",      icon: Wand2,           label: "Skills"      },
  { href: "/developers",  icon: Users,           label: "Developers"  },
  { href: "/agents",      icon: Bot,             label: "Agents"      },
];

export function Sidebar() {
  const path = usePathname();
  return (
    <aside
      className="w-56 shrink-0 flex flex-col"
      style={{
        background:   "var(--cx-sidebar-bg)",
        borderRight:  "1px solid var(--cx-border-subtle)",
        boxShadow:    "var(--cx-shadow-sidebar)",
      }}
    >
      {/* Logo */}
      <div className="px-4 py-5 border-b" style={{ borderColor: "var(--cx-border-subtle)" }}>
        <div className="flex items-center gap-3">
          <Image
            src="/cx-icon.jpg"
            alt="Cx"
            width={40}
            height={40}
            priority
            className="rounded-md shrink-0"
          />
          <Image
            src="/checkmarx-logo.webp"
            alt="Checkmarx"
            width={130}
            height={32}
            priority
            style={{ filter: "brightness(0) saturate(100%) invert(28%) sepia(95%) saturate(600%) hue-rotate(210deg) brightness(90%)" }}
          />
        </div>
        <div className="mt-3 px-1">
          <span
            style={{
              display:         "inline-block",
              background:      "linear-gradient(135deg, #4258c7 0%, #6b7fe0 100%)",
              color:           "#fff",
              fontSize:        "11px",
              fontWeight:      700,
              letterSpacing:   "0.08em",
              textTransform:   "uppercase",
              padding:         "3px 8px",
              borderRadius:    "4px",
            }}
          >
            AI Governance
          </span>
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-2 py-3 space-y-0.5">
        {NAV.map(({ href, icon: Icon, label }) => {
          const active = path === href || (href !== "/" && path.startsWith(href));
          return (
            <Link
              key={href}
              href={href}
              className="flex items-center gap-3 px-3 py-2 rounded text-sm transition-colors"
              style={
                active
                  ? {
                      background:   "#e8ebf8",
                      color:        "var(--cx-primary)",
                      borderLeft:   "3px solid var(--cx-primary)",
                      borderRadius: "0px 4px 4px 0px",
                      paddingLeft:  "calc(0.75rem - 3px)",
                      fontWeight:   600,
                    }
                  : {
                      color:        "#939499",
                      borderLeft:   "3px solid transparent",
                      borderRadius: "0px 4px 4px 0px",
                      paddingLeft:  "calc(0.75rem - 3px)",
                    }
              }
              onMouseEnter={(e) => {
                if (!active) {
                  (e.currentTarget as HTMLElement).style.background = "#f0f2fd";
                  (e.currentTarget as HTMLElement).style.color = "var(--cx-primary-hover)";
                }
              }}
              onMouseLeave={(e) => {
                if (!active) {
                  (e.currentTarget as HTMLElement).style.background = "";
                  (e.currentTarget as HTMLElement).style.color = "#939499";
                }
              }}
            >
              <Icon size={15} style={{ flexShrink: 0 }} />
              <span>{label}</span>
            </Link>
          );
        })}
      </nav>

      {/* Footer */}
      <div
        className="px-4 py-3 border-t text-xs"
        style={{ borderColor: "var(--cx-border-subtle)", color: "#bfbdc2" }}
      >
        Checkmarx AI Governance
      </div>
    </aside>
  );
}
