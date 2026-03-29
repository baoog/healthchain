"use client";

import {
    Dialog,
    DialogTitle,
    DialogContent,
    DialogContentText,
    DialogActions,
    Button,
    Typography,
    Box,
} from "@mui/material";
import CheckCircleIcon from "@mui/icons-material/CheckCircle";
import ErrorIcon from "@mui/icons-material/Error";
import WarningIcon from "@mui/icons-material/Warning";

export type DialogType = "success" | "error" | "warning";

interface CommonConfirmDialogProps {
    open: boolean;
    type?: DialogType;
    title?: string;
    message: string;
    confirmText?: string;
    cancelText?: string;
    onConfirm: () => void;
    onCancel: () => void;
    loading?: boolean;
}

export default function CommonConfirmDialog({
                                                open,
                                                type = "warning",
                                                title,
                                                message,
                                                confirmText = "Confirm",
                                                cancelText = "Cancel",
                                                onConfirm,
                                                onCancel,
                                                loading = false,
                                            }: CommonConfirmDialogProps) {

    const renderIcon = () => {
        switch (type) {
            case "success":
                return <CheckCircleIcon color="success" sx={{ fontSize: 40 }} />;
            case "error":
                return <ErrorIcon color="error" sx={{ fontSize: 40 }} />;
            default:
                return <WarningIcon color="warning" sx={{ fontSize: 40 }} />;
        }
    };

    return (
        <Dialog open={open} onClose={onCancel} maxWidth="xs" fullWidth>
            <DialogTitle>
                <Box display="flex" alignItems="center" gap={1}>
                    {renderIcon()}
                    <Typography variant="h6">
                        {title || type.toUpperCase()}
                    </Typography>
                </Box>
            </DialogTitle>

            <DialogContent>
                <DialogContentText>
                    {message}
                </DialogContentText>
            </DialogContent>

            <DialogActions>
                <Button onClick={onCancel} disabled={loading}>
                    {cancelText}
                </Button>
                <Button
                    variant="contained"
                    color={
                        type === "error"
                            ? "error"
                            : type === "success"
                                ? "success"
                                : "warning"
                    }
                    onClick={onConfirm}
                    disabled={loading}
                >
                    {confirmText}
                </Button>
            </DialogActions>
        </Dialog>
    );
}