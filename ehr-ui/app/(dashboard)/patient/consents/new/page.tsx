"use client";

import {
    Alert,
    Autocomplete,
    Box,
    Button,
    Card,
    CardContent,
    Checkbox,
    CircularProgress,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    Divider,
    FormControlLabel,
    Grid,
    Paper,
    Snackbar,
    TextField,
    Typography,
} from "@mui/material";
import React, { useEffect, useMemo, useState } from "react";
import { Html5QrcodeScanner } from "html5-qrcode";
import { api } from "@/lib/api";
import { useAuth } from "@/context/AuthContext";
import {BundleType} from "@/lib/bundle";


type BundleListItem = {
    id: string;
    type: BundleType;
    created_at: string;
    owner?: string;
    sha256?: string;
    uri?: string;
};

type DoctorOption = {
    id?: string;
    name: string;
    national_id_hash: string; // this is what backend expects for granting
    department?: string;
};

function decodeDoctorQRText(raw: string): { doctorHash?: string; userId?: string } {
    // Accept formats:
    // - "HC:<base64>"
    // - "<base64>"
    const trimmed = (raw || "").trim();
    const base64 = trimmed.startsWith("HC:") ? trimmed.slice(3) : trimmed;

    const jsonStr = atob(base64);
    const payload = JSON.parse(jsonStr);

    // You said you encode like:
    // { type:"DOCTOR_QR", doctorHash, issuedAt }
    // but we also support { userId } if you change later.
    return {
        doctorHash: payload?.doctorHash,
        userId: payload?.userId,
    };
}

export default function PatientGrantConsentPage() {
    const { role } = useAuth();

    // Only allow PATIENT
    if (role && role !== "PATIENT") return null;

    // Doctor selection
    const [doctorQuery, setDoctorQuery] = useState("");
    const [doctorOptions, setDoctorOptions] = useState<DoctorOption[]>([]);
    const [doctorLoading, setDoctorLoading] = useState(false);
    const [doctorSelected, setDoctorSelected] = useState<DoctorOption | null>(null);

    // Bundles
    const [bundles, setBundles] = useState<BundleListItem[]>([]);
    const [loadingBundles, setLoadingBundles] = useState(false);
    const [selected, setSelected] = useState<Record<string, boolean>>({});
    const [selectAll, setSelectAll] = useState(false);

    // Optional expiry
    const [expiresAt, setExpiresAt] = useState<string>("");

    // Snackbar
    const [snack, setSnack] = useState<{ open: boolean; msg: string; severity: "success" | "error" | "info" }>({
        open: false,
        msg: "",
        severity: "info",
    });

    // QR scanner dialog state
    const [scanOpen, setScanOpen] = useState(false);

    const selectedBundleIds = useMemo(
        () => Object.entries(selected).filter(([, v]) => v).map(([k]) => k),
        [selected]
    );

    const effectiveDoctorID = useMemo(() => doctorSelected?.national_id_hash || "", [doctorSelected]);
    const effectiveDoctorHash = useMemo(() => doctorSelected?.national_id_hash || "", [doctorSelected]);

    async function loadMyBundles() {
        setLoadingBundles(true);
        try {
            const res = await api.get("/healthchain/v1/fabric/bundles");
            const items: BundleListItem[] =
                Array.isArray(res.data) ? res.data : res.data?.result || res.data?.bundles || [];
            setBundles(items);

            const initial: Record<string, boolean> = {};
            items.forEach((b) => (initial[b.id] = false));
            setSelected(initial);
            setSelectAll(false);
        } catch (e: any) {
            setSnack({
                open: true,
                msg: e?.response?.data?.error || e?.message || "Failed to load bundles",
                severity: "error",
            });
        } finally {
            setLoadingBundles(false);
        }
    }

    useEffect(() => {
        loadMyBundles();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    useEffect(() => {
        if (!bundles.length) return;
        const allChecked = bundles.every((b) => selected[b.id] === true);
        setSelectAll(allChecked);
    }, [selected, bundles]);

    function toggleSelectAll(checked: boolean) {
        const next: Record<string, boolean> = {};
        bundles.forEach((b) => (next[b.id] = checked));
        setSelected(next);
        setSelectAll(checked);
    }

    function toggleOne(id: string, checked: boolean) {
        setSelected((prev) => ({ ...prev, [id]: checked }));
    }

    async function searchDoctors(name: string) {
        setDoctorLoading(true);
        try {
            // adjust to your real API
            const res = await api.get("/api/v1/user/doctor/search-by-name", {
                params: { pageSize: "20", name },
            });
            const items: DoctorOption[] = Array.isArray(res.data) ? res.data : res.data?.result || [];
            setDoctorOptions(items);
        } catch (e: any) {
            setDoctorOptions([]);
            setSnack({
                open: true,
                msg: e?.response?.data?.error || e?.message || "Doctor search failed (check API endpoint)",
                severity: "error",
            });
        } finally {
            setDoctorLoading(false);
        }
    }

    useEffect(() => {
        const q = doctorQuery.trim();
        if (q.length < 2) {
            setDoctorOptions([]);
            return;
        }
        const t = setTimeout(() => searchDoctors(q), 350);
        return () => clearTimeout(t);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [doctorQuery]);

    function handleScanResult(text: string) {
        try {
            const decoded = decodeDoctorQRText(text);
            // Prefer doctorHash, fallback to userId if your QR carries that
            const dh = decoded.doctorHash || decoded.userId;
            if (!dh) throw new Error("QR payload missing doctorHash/userId.");
            setDoctorSelected({ name: "Scanned Doctor", national_id_hash: dh });
            setSnack({ open: true, msg: "Doctor QR scanned successfully.", severity: "success" });
            setScanOpen(false);
        } catch (err: any) {
            setSnack({ open: true, msg: err?.message || "Invalid QR code.", severity: "error" });
        }
    }


    // QRScanner component (html5-qrcode)

function QRScanner({ onResult }: { onResult: (text: string) => void }) {
    useEffect(() => {
        const scanner = new Html5QrcodeScanner(
            "qr-reader",
            { fps: 10, qrbox: 250 },
            false
        );

        scanner.render(
            (decodedText) => {
                onResult(decodedText);
                scanner.clear();
            },
            () => {}
        );

        return () => {
            scanner.clear().catch(() => {});
        };
    }, [onResult]);

    return <div id="qr-reader" style={{ width: "100%" }} />;
}

    async function grantConsent() {
        const dh = effectiveDoctorID;
        if (!dh) {
            setSnack({ open: true, msg: "Please select a doctor first.", severity: "error" });
            return;
        }
        if (!selectedBundleIds.length) {
            setSnack({ open: true, msg: "Please select at least one bundle.", severity: "error" });
            return;
        }

        try {
            const expiresRFC3339 = expiresAt ? new Date(expiresAt).toISOString() : undefined;

            await api.post("/healthchain/v1/consents/grant", {
                doctor_id: dh,
                bundle_ids: selectedBundleIds,
                expires_at: expiresRFC3339,
            });

            setSnack({ open: true, msg: "Consent granted successfully.", severity: "success" });
            toggleSelectAll(false);
            setExpiresAt("");
        } catch (e: any) {
            setSnack({
                open: true,
                msg: e?.response?.data?.error || e?.message || "Grant consent failed",
                severity: "error",
            });
        }
    }

    return (
        <Box>
            <Typography variant="h5" fontWeight={700} color="black" gutterBottom>
                Grant Consent
            </Typography>

            <Typography color="text.secondary" mb={2}>
                Search a doctor by name, or scan the doctor QR code using your camera. Then select which bundles you want to share.
            </Typography>

            <Grid container spacing={2}>
                {/* LEFT: Doctor selection */}
                <Grid item size={{xs:12, md:12}}>
                    <Card>
                        <CardContent>
                            <Typography fontWeight={700} mb={1}>
                                Step 1 — Choose Doctor
                            </Typography>

                            <Box display="flex" gap={1} alignItems="flex-start">
                                <Autocomplete
                                    sx={{ flex: 1 }}
                                    options={doctorOptions}
                                    loading={doctorLoading}
                                    value={doctorSelected}
                                    onChange={(_, v) => setDoctorSelected(v)}
                                    inputValue={doctorQuery}
                                    onInputChange={(_, v) => setDoctorQuery(v)}
                                    getOptionLabel={(o) => `${o.name}${o.department ? ` — ${o.department}` : ""}`}
                                    renderInput={(params) => (
                                        <TextField
                                            {...params}
                                            label="Search doctor"
                                            placeholder="Type doctor name..."
                                            InputProps={{
                                                ...params.InputProps,
                                                endAdornment: (
                                                    <>
                                                        {doctorLoading ? <CircularProgress size={18} /> : null}
                                                        {params.InputProps.endAdornment}
                                                    </>
                                                ),
                                            }}
                                        />
                                    )}
                                />

                                <Button
                                    variant="contained"
                                    onClick={() => setScanOpen(true)}
                                    sx={{ height: 56, whiteSpace: "nowrap" }}
                                >
                                    Scan QR
                                </Button>
                            </Box>

                            <Divider sx={{ my: 2 }} />

                            <Typography variant="body2" color="text.secondary">
                                Doctor Hash:
                            </Typography>
                            <Paper variant="outlined" sx={{ p: 1, mt: 1, wordBreak: "break-all" }}>
                                <Typography fontFamily="monospace" fontSize={13}>
                                    {effectiveDoctorHash || "-"}
                                </Typography>
                            </Paper>
                        </CardContent>
                    </Card>
                </Grid>

                {/* RIGHT: Bundle selection */}
                <Grid item size={{xs:12, md:12}}>
                    <Card>
                        <CardContent>
                            <Typography fontWeight={700} mb={1}>
                                Step 2 — Select Bundles
                            </Typography>

                            <Box display="flex" alignItems="center" justifyContent="space-between" gap={2} mb={2} flexWrap="wrap">
                                <FormControlLabel
                                    control={<Checkbox checked={selectAll} onChange={(e) => toggleSelectAll(e.target.checked)} />}
                                    label="Select all"
                                />

                                <Button variant="outlined" onClick={loadMyBundles} disabled={loadingBundles}>
                                    Refresh bundles
                                </Button>
                            </Box>

                            {loadingBundles ? (
                                <Box display="flex" justifyContent="center" py={4}>
                                    <CircularProgress />
                                </Box>
                            ) : (
                                <Paper variant="outlined" sx={{ p: 1 }}>
                                    {bundles.length === 0 ? (
                                        <Typography color="text.secondary" p={2}>
                                            No bundles found.
                                        </Typography>
                                    ) : (
                                        <Box>
                                            {bundles.map((b) => (
                                                <Box
                                                    key={b.id}
                                                    sx={{
                                                        display: "flex",
                                                        alignItems: "center",
                                                        gap: 1,
                                                        p: 1,
                                                        borderBottom: "1px solid rgba(0,0,0,0.08)",
                                                    }}
                                                >
                                                    <Checkbox checked={!!selected[b.id]} onChange={(e) => toggleOne(b.id, e.target.checked)} />
                                                    <Box flex={1}>
                                                        <Typography fontWeight={600} color="black">
                                                            {b.type}
                                                        </Typography>
                                                        <Typography variant="caption" color="text.secondary">
                                                            {b.id} • {new Date(b.created_at).toLocaleString()}
                                                        </Typography>
                                                    </Box>
                                                </Box>
                                            ))}
                                        </Box>
                                    )}
                                </Paper>
                            )}

                            <Divider sx={{ my: 2 }} />

                            <Typography fontWeight={700} mb={1}>
                                Step 3 — Confirm Grant
                            </Typography>

                            <TextField
                                label="Expiry (optional)"
                                type="datetime-local"
                                value={expiresAt}
                                onChange={(e) => setExpiresAt(e.target.value)}
                                fullWidth
                                InputLabelProps={{ shrink: true }}
                                sx={{ mb: 2 }}
                            />

                            <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
                                <Typography variant="body2" color="text.secondary">
                                    Selected bundles:
                                </Typography>
                                <Typography fontFamily="monospace" fontSize={13} sx={{ wordBreak: "break-all" }}>
                                    {selectedBundleIds.length ? selectedBundleIds.join(", ") : "-"}
                                </Typography>

                                <Divider sx={{ my: 1.5 }} />

                                <Typography variant="body2" color="text.secondary">
                                    Doctor hash:
                                </Typography>
                                <Typography fontFamily="monospace" fontSize={13} sx={{ wordBreak: "break-all" }}>
                                    {effectiveDoctorHash || "-"}
                                </Typography>
                            </Paper>

                            <Button variant="contained" fullWidth onClick={grantConsent}>
                                Grant Consent
                            </Button>
                        </CardContent>
                    </Card>
                </Grid>
            </Grid>

            {/* QR Scan Dialog */}
            <Dialog open={scanOpen} onClose={() => setScanOpen(false)} fullWidth maxWidth="sm">
                <DialogTitle>Scan Doctor QR</DialogTitle>
                <DialogContent>
                    <Alert severity="info" sx={{ mb: 2 }}>
                        Point the camera at the doctor QR code.
                    </Alert>
                    <QRScanner
                        onResult={(text) => {
                            handleScanResult(text);
                            setScanOpen(false);
                        }}
                    />
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setScanOpen(false)}>Close</Button>
                </DialogActions>
            </Dialog>

            <Snackbar
                open={snack.open}
                autoHideDuration={3500}
                onClose={() => setSnack((s) => ({ ...s, open: false }))}
                anchorOrigin={{ vertical: "bottom", horizontal: "center" }}
            >
                <Alert severity={snack.severity} onClose={() => setSnack((s) => ({ ...s, open: false }))} variant="filled">
                    {snack.msg}
                </Alert>
            </Snackbar>
        </Box>
    );
}