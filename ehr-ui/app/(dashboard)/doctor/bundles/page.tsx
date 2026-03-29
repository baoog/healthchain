"use client";

import { useState } from "react";
import {
    Box,
    Button,
    MenuItem,
    Select,
    TextField,
    Typography,
    Paper,
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableRow,
    CircularProgress, Chip,
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions
} from "@mui/material";
import { COUNTRIES } from "@/lib/countries";
import { sha256 } from "@/lib/hash";
import {api} from "@/lib/api";
import { useRouter } from "next/navigation";
import {formatDate} from "@/lib/util";
import AddIcon from "@mui/icons-material/Add";

interface Bundle {
    id: string;
    type: string;
    user?: User;
    created_at: string;
    is_external?: boolean
}

interface User {
    name: string;
}

export default function DoctorBundlesPage() {
    const [country, setCountry] = useState("");
    const [nationalId, setNationalId] = useState("");
    const [loading, setLoading] = useState(false);
    const [bundles, setBundles] = useState<Bundle[]>([]);
    const [nextPageToken, setNextPageToken] = useState<string | null>(null);
    const [error, setError] = useState<string | null>(null);

    const canSearch = country && nationalId;

    const [openNotFoundDialog, setOpenNotFoundDialog] = useState(false);
    const [openRegisterDialog, setOpenRegisterDialog] = useState(false);
    const [registerName, setRegisterName] = useState("");
    const [registerCountry, setRegisterCountry] = useState("");
    const [registerNationalId, setRegisterNationalId] = useState("");
    const [registerLoading, setRegisterLoading] = useState(false);

    const handleRegister = async () => {
        try {
            setRegisterLoading(true);

            await api.post("/api/v1/user", {
                role: "PATIENT",
                country_code: country,
                national_id: nationalId,
                name: registerName
            });

            setOpenRegisterDialog(false);
            setOpenNotFoundDialog(false);

            // auto search again after registration
            handleSearch();

        } catch (err: any) {
            setError(err.message);
        } finally {
            setRegisterLoading(false);
        }
    };

    const handleSearch = async () => {
        try {
            setLoading(true);
            setError(null);
            setBundles([]);
            setNextPageToken(null);

            const patientHash = await sha256(`${country}:${nationalId}`);
            const res = await api.get(`/api/v1/bundles/granted?patientHash=${patientHash}`);
            const {result, ok} = res.data;
            if(ok) {
                setBundles(result.bundles || []);
                setNextPageToken(result.next_page_token || null);
            }

        } catch (e: any) {
            console.log(e?.response?.data)
                if (e?.response?.data?.error_code === "USER_NOT_FOUND") {
                    setOpenNotFoundDialog(true);
                } else {
                    setError(e.message);
                }

        } finally {
            setLoading(false);
        }
    };

    const loadMore = async () => {
        if (!nextPageToken) return;
        setLoading(true);

        const patientHash = await sha256(`${country}:${nationalId}`);
        const res = await api.get(`/api/v1/bundles/granted?patientHash=${patientHash}&${nextPageToken}`);
        const {result, ok} = res.data;
        if(ok) {
            setBundles((prev) => [...prev, ...(result.bundles || [])]);
            setNextPageToken(result.nextPageToken || null);
            setLoading(false);
        }

    };
    const router = useRouter();


    return (
        <Box sx={{display: "flex",flexDirection: "column"}}>
            <Typography variant="h5" fontWeight={700} mb={2} color={"black"}>
                My Bundles
            </Typography>

            {/* Search form */}
            <Paper sx={{ p: 2, mb: 3 }}>
                <Box display="flex" gap={2} flexWrap="wrap" justifyContent={"space-between"} >
                    <Box display="flex" gap={2} flexWrap="wrap" >
                        <Select
                            value={country}
                            onChange={(e) => setCountry(e.target.value)}
                            displayEmpty
                            sx={{ minWidth: 200 }}
                        >
                            <MenuItem value="" disabled>
                                Country Code
                            </MenuItem>
                            {COUNTRIES.map((c) => (
                                <MenuItem key={c.code} value={c.code}>
                                    {c.label} ({c.code})
                                </MenuItem>
                            ))}
                        </Select>

                        <TextField
                            label="National ID"
                            value={nationalId}
                            onChange={(e) => setNationalId(e.target.value)}
                        />

                        <Button
                            variant="contained"
                            disabled={!canSearch || loading}
                            onClick={handleSearch}
                        >
                            Search
                        </Button>
                    </Box>

                    <Button variant="contained"
                            startIcon={<AddIcon />}
                            onClick={() => router.push("/doctor/bundles/new")}>Bundle</Button>
                </Box>
            </Paper>

            {/* Error */}
            {error && (
                <Typography color="error" mb={2}>
                    {error}
                </Typography>
            )}

            {/* Table */}
            <Paper>
                <Table>
                    <TableHead>
                        <TableRow>
                            <TableCell>Bundle ID</TableCell>
                            <TableCell>Patient's Name</TableCell>
                            <TableCell>External</TableCell>
                            <TableCell>Type</TableCell>
                            <TableCell>Created At</TableCell>
                        </TableRow>
                    </TableHead>

                    <TableBody>
                        {bundles.map((b) => (
                            <TableRow key={b.id}
                                      hover
                                      sx={{ cursor: "pointer" }}
                                      onClick={() => router.push(`/bundles/${b.id}`)}>
                                <TableCell>{b.id}</TableCell>
                                <TableCell>{b.user?.name}</TableCell>
                                <TableCell>
                                    {b.is_external ? (
                                        <Chip label="Yes" color="warning" size="small" />
                                    ) : (
                                        <Chip label="No" color="primary" size="small" />
                                    )}
                                </TableCell>
                                <TableCell>{b.type}</TableCell>
                                <TableCell>{formatDate(b.created_at)}</TableCell>
                            </TableRow>

                        ))}
                    </TableBody>
                </Table>

                {loading && (
                    <Box p={2} textAlign="center">
                        <CircularProgress size={24} />
                    </Box>
                )}

                {nextPageToken && !loading && (
                    <Box p={2} textAlign="center">
                        <Button onClick={loadMore}>Load more</Button>
                    </Box>
                )}
            </Paper>
            <Dialog open={openNotFoundDialog} onClose={() => setOpenNotFoundDialog(false)}>
                <DialogTitle>Patient Not Found</DialogTitle>
                <DialogContent>
                    <Typography>
                        Looks like this patient isn’t registered yet.
                        Want to create an account now?
                    </Typography>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setOpenNotFoundDialog(false)}>
                        Cancel
                    </Button>
                    <Button
                        variant="contained"
                        onClick={() => {
                            setOpenNotFoundDialog(false);
                            setOpenRegisterDialog(true);
                        }}
                    >
                        Register
                    </Button>
                </DialogActions>
            </Dialog>
            <Dialog open={openRegisterDialog} onClose={() => setOpenRegisterDialog(false)} maxWidth="sm" fullWidth>
                <DialogTitle>Register New Patient</DialogTitle>

                <DialogContent>
                    <Box mt={1} display="flex" flexDirection="column" gap={2}>

                        {/* Country Code */}
                        <Select
                            value={registerCountry || country}
                            onChange={(e) => setRegisterCountry(e.target.value)}
                            displayEmpty
                            fullWidth
                        >
                            <MenuItem value="" disabled>
                                Select Country Code
                            </MenuItem>
                            {COUNTRIES.map((c) => (
                                <MenuItem key={c.code} value={c.code}>
                                    {c.label} ({c.code})
                                </MenuItem>
                            ))}
                        </Select>

                        {/* Citizen ID */}
                        <TextField
                            label="Citizen ID"
                            value={registerNationalId || nationalId}
                            onChange={(e) => setRegisterNationalId(e.target.value)}
                            fullWidth
                        />

                        {/* Full Name */}
                        <TextField
                            label="Full Name"
                            value={registerName}
                            onChange={(e) => setRegisterName(e.target.value)}
                            fullWidth
                        />

                    </Box>
                </DialogContent>

                <DialogActions>
                    <Button onClick={() => setOpenRegisterDialog(false)}>
                        Cancel
                    </Button>

                    <Button
                        variant="contained"
                        disabled={!country || !nationalId || !registerName || registerLoading}
                        onClick={handleRegister}
                    >
                        {registerLoading ? "Registering..." : "Register"}
                    </Button>
                </DialogActions>
            </Dialog>
        </Box>
    );
}