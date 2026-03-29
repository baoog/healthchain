"use client";

import {
    Box,
    IconButton,
    Paper,
    Stack,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    Typography,
    Chip,
} from "@mui/material";
import CheckCircleIcon from "@mui/icons-material/CheckCircle";
import CancelIcon from "@mui/icons-material/Cancel";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import {ellipsisCell, formatDate} from "@/lib/util";
import CommonConfirmDialog from "@/components/Modal";

interface BreakGlassRequest {
    request_id: string;
    owner_hash: string;
    doctor_hash: string;
    justification: string;
    status: string;
    expires_at: string;
    created_at: string;
    is_external: boolean;
}

export default function PendingBreakGlassPage() {
    const router = useRouter();
    const [requests, setRequests] = useState<BreakGlassRequest[]>([]);
    const [loading, setLoading] = useState(false);

    const [openDialog, setOpenDialog] = useState(false);
    const [dialogType, setDialogType] = useState<"success" | "error" | "warning">("warning");
    const [dialogMessage, setDialogMessage] = useState("");

    const loadPending = async () => {
        setLoading(true);
        try {
            const res = await api.get(
                "/healthchain/v1/consents/emergency/request/pending"
            );

            setRequests(Array.isArray(res.data.result) ? res.data.result : []);
        } catch (err) {
            console.error("Failed to load pending break-glass requests", err);
            setRequests([]);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        if(!requests) return;
        loadPending();
    }, []);

    const updateStatus = async (
        requestID: string,
        status: "APPROVED" | "REJECTED"
    ) => {
        if (
            !confirm(
                `Are you sure you want to ${status.toLowerCase()} this request?`
            )
        )
            return;

        try {
            await api.post(
                "/healthchain/v1/consents/emergency/request/approve",
                {
                    request_id: requestID,
                    decision: status,
                }
            );
            await loadPending();

        } catch (err) {
            setDialogType("error");
            setDialogMessage(`Something went wrong. Can't update status: ${err}`);
            setOpenDialog(true);
            console.error(err);
        }
    };

    return (
        <Box p={3}>
            <Typography variant="h5" fontWeight={600} mb={2} color="black">
                Pending Break-Glass Requests
            </Typography>

            <TableContainer component={Paper}>
                <Table>
                    <TableHead>
                        <TableRow>
                            <TableCell>Request ID</TableCell>
                            <TableCell>Doctor</TableCell>
                            <TableCell>Justification</TableCell>
                            <TableCell>External</TableCell>
                            <TableCell>Expires At</TableCell>
                            <TableCell>Status</TableCell>
                            <TableCell align="center">Action</TableCell>
                        </TableRow>
                    </TableHead>

                    <TableBody>
                        {!loading && requests.length === 0 && (
                            <TableRow>
                                <TableCell colSpan={7} align="center">
                                    No pending requests
                                </TableCell>
                            </TableRow>
                        )}

                        {requests.map((r) => (
                            <TableRow
                                key={r.request_id}
                                hover
                                sx={{ cursor: "pointer" }}
                                onClick={() =>
                                    router.push(`/break-glass/${r.request_id}`)
                                }
                            >
                                <TableCell sx={ellipsisCell}>{r.request_id}</TableCell>
                                <TableCell>{r.doctor_hash}</TableCell>
                                <TableCell>{r.justification}</TableCell>
                                <TableCell>
                                    {r.is_external ? (
                                        <Chip label="External" color="warning" size="small" />
                                    ) : (
                                        <Chip label="Internal" color="primary" size="small" />
                                    )}
                                </TableCell>
                                <TableCell>
                                    {formatDate(r.expires_at)}
                                </TableCell>
                                <TableCell>
                                    <Chip label={r.status} color={r.status === "REQUESTED" ? "primary" : r.status === "EXPIRED" ||r.status === "REJECTED" ?"error" :  "warning"} size="small" />
                                </TableCell>

                                <TableCell align="center" onClick={(e) => e.stopPropagation()}>
                                    <Stack direction="row" spacing={1} justifyContent="center">
                                        <IconButton
                                            color="success"
                                            onClick={() =>
                                                updateStatus(r.request_id, "APPROVED")
                                            }
                                        >
                                            <CheckCircleIcon />
                                        </IconButton>

                                        <IconButton
                                            color="error"
                                            onClick={() =>
                                                updateStatus(r.request_id, "REJECTED")
                                            }
                                        >
                                            <CancelIcon />
                                        </IconButton>
                                    </Stack>
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </TableContainer>
            <CommonConfirmDialog
                open={openDialog}
                type={dialogType}
                message={dialogMessage}
                onConfirm={() => setOpenDialog(false)}
                onCancel={() => setOpenDialog(false)}
            />
        </Box>
    );
}