"use client";

import { Box, Paper, Typography, Divider, Button } from "@mui/material";
import QRCode from "react-qr-code";
import { useAuth } from "@/context/AuthContext";
import { useEffect, useState } from "react";
import RefreshIcon from "@mui/icons-material/Refresh";

export default function DoctorQRCodePage() {
    const { userToken, role } = useAuth();

    // ✅ Hooks MUST be called unconditionally
    const [qrValue, setQrValue] = useState<string>("");

    const buildQRValue = () => {
        if (!userToken) return "";

        const payload = {
            type: "DOCTOR_QR",
            doctorHash: userToken.fabricId,
            issuedAt: Math.floor(Date.now() / 1000),
        };

        return btoa(JSON.stringify(payload));
    };

    // ✅ Generate QR when user becomes available
    useEffect(() => {
        if (role === "DOCTOR" && userToken) {
            setQrValue(buildQRValue());
        }
    }, [role, userToken]);

    const rotateQRCode = () => {
        setQrValue(buildQRValue());
    };

    // ✅ Safe conditional rendering AFTER hooks
    if (role !== "DOCTOR") return null;
    if (!qrValue) return null;

    return (
        <Box>
            <Typography variant="h5" fontWeight={600} gutterBottom color={"black"}>
                My QR Code
            </Typography>

            <Typography color="text.secondary" mb={2}>
                Patients can scan this QR code to grant you access to their health bundles.
            </Typography>

            <Paper
                elevation={3}
                sx={{
                    p: 4,
                    textAlign: "center",
                }}
            >
                <Box
                    sx={{
                        display: "flex",
                        justifyContent: "center",
                        alignItems: "center",
                        minHeight: 260,
                    }}
                >
                    <QRCode value={qrValue} size={220} />
                </Box>

                <Divider sx={{ my: 2 }} />

                <Button
                    variant="outlined"
                    startIcon={<RefreshIcon />}
                    onClick={rotateQRCode}

                >
                    Rotate QR Code
                </Button>

                <Typography
                    variant="caption"
                    color="text.secondary"
                    display="block"
                    mt={1}
                >
                    Generates a new temporary QR code
                </Typography>
            </Paper>
        </Box>
    );
}