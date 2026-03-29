"use client";

import React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
    Drawer,
    List,
    ListItemButton,
    ListItemIcon,
    ListItemText,
    Box,
    useMediaQuery,
    useTheme,
} from "@mui/material";
import { Dashboard, Folder, Shield, Warning } from "@mui/icons-material";
import { useAuth } from "@/context/AuthContext";
import { TOPBAR_HEIGHT } from "@/components/Topbar";

type UserRole = "PATIENT" | "DOCTOR" | "HOD";

interface MenuItem {
    label: string;
    icon: React.ReactNode;
    href: string;
}

const MENU_BY_ROLE: Record<UserRole, MenuItem[]> = {
    PATIENT: [
        { label: "Dashboard", icon: <Dashboard />, href: "/patient" },
        { label: "My Bundles", icon: <Folder />, href: "/patient/bundles" },
        { label: "Consents", icon: <Shield />, href: "/patient/consents" },
    ],
    DOCTOR: [
        { label: "My Bundles", icon: <Folder />, href: "/doctor/bundles" },
        { label: "Consents", icon: <Shield />, href: "/doctor/consents" },
        { label: "Break-Glass", icon: <Warning />, href: "/doctor/break-glass" },
    ],
    HOD: [{ label: "Break-Glass", icon: <Warning />, href: "/hod/break-glass" }],
};

interface SidebarProps {
    collapsed: boolean;
    mobileOpen: boolean;
    onCloseMobile: () => void;
}

export const SIDEBAR_EXPANDED = 340;
export const SIDEBAR_COLLAPSED = 72;

export default function Sidebar({ collapsed, mobileOpen, onCloseMobile }: SidebarProps) {
    const theme = useTheme();
    const isDesktop = useMediaQuery(theme.breakpoints.up("md"));
    const pathname = usePathname();

    const { role } = useAuth();
    if (!role) return null;

    const normalizedRole = (role || "").toUpperCase() as UserRole;
    const safeRole: UserRole = normalizedRole in MENU_BY_ROLE ? normalizedRole : "PATIENT";

    // ✅ Mobile drawer must always show ICON + LABEL even if desktop is collapsed
    const effectiveCollapsed = isDesktop ? collapsed : false;

    const drawerWidth = effectiveCollapsed ? SIDEBAR_COLLAPSED : SIDEBAR_EXPANDED;
    const items = MENU_BY_ROLE[safeRole];

    const content = (
        <Box
          sx={{
            height: "100%",
            background: "linear-gradient(180deg, #02D8E9 0%, #1b0f3b 100%)",
            color: "#fff",
            display: "flex",
            flexDirection: "column",
          }}
        >
          <Box
            sx={{
              height: 64,
              display: "flex",
              alignItems: "center",
              px: effectiveCollapsed ? 2 : 3,
              fontWeight: 700,
              fontSize: 18,
              letterSpacing: 0.5,
            }}
          >
            {!effectiveCollapsed ? "HealthChain" : "HC"}
          </Box>

          <List>
            {items.map((item) => (
              <ListItemButton
                key={item.label}
                component={Link}
                href={item.href}
                selected={pathname === item.href}
                sx={{
                  justifyContent: effectiveCollapsed ? "center" : "flex-start",
                  px: effectiveCollapsed ? 1.5 : 3,
                  minHeight: 48,
                  color: "#fff",
                  "&.Mui-selected": { backgroundColor: "rgba(255,255,255,0.12)" },
                  "&.Mui-selected:hover": { backgroundColor: "rgba(255,255,255,0.18)" },
                }}
              >
                <ListItemIcon
                  sx={{
                    color: "#fff",
                    minWidth: effectiveCollapsed ? 0 : 40,
                    justifyContent: "center",
                  }}
                >
                  {item.icon}
                </ListItemIcon>

                {!effectiveCollapsed && (
                  <ListItemText
                    primary={item.label}
                    primaryTypographyProps={{ fontSize: 14, color: "#fff" }}
                  />
                )}
              </ListItemButton>
            ))}
          </List>
        </Box>
    );

    return (
        <>
            {/* Desktop */}
            <Drawer
                variant="permanent"
                open
                sx={{
                    display: { xs: "none", md: "block" },
                    "& .MuiDrawer-paper": {
                        width: drawerWidth,
                        transition: "width 0.25s",
                        overflowX: "hidden",
                        borderRight: "none",
                        position: "fixed",
                        top: `0px`,
                        height: `100%`,
                    },
                }}
            >
                {content}
            </Drawer>

            {/* Mobile (click outside closes by default) */}
            <Drawer
                variant="temporary"
                open={mobileOpen}
                onClose={onCloseMobile}
                ModalProps={{ keepMounted: true }}
                sx={{
                    display: { xs: "block", md: "none" },
                    "& .MuiDrawer-paper": {
                        width: 240,
                        // top: `${TOPBAR_HEIGHT}px`,
                        height: `100%`,
                        position: "fixed",
                        overflowX: "hidden",
                        borderRight: "none",
                        zIndex: (theme) => theme.zIndex.drawer + 3,
                    },
                }}
            >
                {content}
            </Drawer>
        </>
    );
}