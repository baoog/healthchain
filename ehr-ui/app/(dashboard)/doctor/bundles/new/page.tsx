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
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    CircularProgress, Stack
} from "@mui/material";
import { COUNTRIES } from "@/lib/countries";
import { sha256 } from "@/lib/hash";
import { api } from "@/lib/api";
import { useRouter } from "next/navigation";

type Step = "SEARCH" | "FORM";

export default function CreateBundlePage() {
    const router = useRouter();

    const [step, setStep] = useState<Step>("SEARCH");
    const [form, setForm] = useState<any>({})

    const [country, setCountry] = useState("");
    const [citizenId, setCitizenId] = useState("");
    const [patient, setPatient] = useState<any>(null);

    const [bundleType, setBundleType] = useState("");
    const [content, setContent] = useState("");

    const [loading, setLoading] = useState(false);
    const [openModal, setOpenModal] = useState(false);

    const canSearch = country && citizenId;
    const canSubmit = bundleType && form;

    // ===============================
    // Step 1: Search patient
    // ===============================
    const handleSearch = async () => {
        try {
            setLoading(true);

            const patientHash = await sha256(`${country}:${citizenId}`);

            const res = await api.get(
                `/api/v1/user/patient/get-by-hash?hash=${patientHash}`
            );

            const { ok, result } = res.data;

            if (ok && result) {
                setPatient(result);
                setStep("FORM");
            } else {
                setOpenModal(true);
            }
        } catch (e) {
            setOpenModal(true);
        } finally {
            setLoading(false);
        }
    };

    const handleChange = (key: string, value: any) => {
        setForm((prev: any) => ({ ...prev, [key]: value }));
    };

    // ===============================
    // Create bundle
    // ===============================
    const handleCreateBundle = async () => {
        try {
            setLoading(true);

            const patientHash = await sha256(`${country}:${citizenId}`);

            await api.post("/healthchain/v1/bundles", {
                type: bundleType,
                owner: patientHash,
                data: form,
            });

            router.push("/doctor/bundles");
        } catch (e) {
            console.error(e);
        } finally {
            setLoading(false);
        }
    };

    return (
        <Box p={2}>
            <Typography variant="h5" fontWeight={700} mb={3} color={"black"}>
                Create Bundle
            </Typography>

            {/* ================= SEARCH STEP ================= */}
            {step === "SEARCH" && (
                <Paper sx={{ p: 3 }}>
                    <Typography mb={2} fontWeight={600}>
                        Step 1: Search Patient
                    </Typography>

                    <Box display="flex" gap={2} flexWrap="wrap">
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
                            label="Citizen ID"
                            value={citizenId}
                            onChange={(e) => setCitizenId(e.target.value)}
                        />

                        <Button
                            variant="contained"
                            disabled={!canSearch || loading}
                            onClick={handleSearch}
                        >
                            {loading ? <CircularProgress size={20} /> : "Search"}
                        </Button>
                    </Box>
                </Paper>
            )}

            {/* ================= FORM STEP ================= */}
            {step === "FORM" && (
                <Paper sx={{ p: 3 }}>
                    <Typography mb={2} fontWeight={600}>
                        Step 2: Create Bundle
                    </Typography>

                    <Typography mb={2}>
                        Patient: <strong>{patient?.name}</strong>
                    </Typography>

                    <Box display="flex" flexDirection="column" gap={2}>
                        <Select
                            value={bundleType}
                            onChange={(e) => setBundleType(e.target.value)}
                            displayEmpty
                        >
                            <MenuItem value="" disabled>
                                Select Type
                            </MenuItem>
                            <MenuItem value="ALLERGY">Allergy</MenuItem>
                            <MenuItem value="MEDICATION">Medication</MenuItem>
                            <MenuItem value="BLOOD_GROUP">Blood Group</MenuItem>
                            <MenuItem value="EMERGENCY_SUMMARY">
                                Emergency Summary
                            </MenuItem>
                        </Select>

                        {/* Dynamic form */}
                        {bundleType === "ALLERGY" && (
                            <Stack spacing={2}>
                                <TextField label="Allergen" onChange={(e) => handleChange("allergen", e.target.value)} />
                                <TextField label="Reaction" onChange={(e) => handleChange("reaction", e.target.value)} />
                                <TextField label="Severity" onChange={(e) => handleChange("severity", e.target.value)} />
                                <TextField type="date" label="Onset Date" InputLabelProps={{ shrink: true }}
                                           onChange={(e) => handleChange("onsetDate", e.target.value)} />
                                <TextField label="Notes" multiline rows={3}
                                           onChange={(e) => handleChange("notes", e.target.value)} />
                            </Stack>
                        )}

                        {bundleType === "MEDICATION" && (
                            <Stack spacing={2}>
                                <TextField label="Drug Name" onChange={(e) => handleChange("drug_name", e.target.value)} />
                                <TextField label="Dose" onChange={(e) => handleChange("dose", e.target.value)} />
                                <TextField label="Frequency" onChange={(e) => handleChange("frequency", e.target.value)} />
                                <TextField label="Route" onChange={(e) => handleChange("route", e.target.value)} />
                                <TextField type="date" label="Start Date" InputLabelProps={{ shrink: true }}
                                           onChange={(e) => handleChange("start_date", e.target.value)} />
                                <TextField type="date" label="End Date" InputLabelProps={{ shrink: true }}
                                           onChange={(e) => handleChange("end_date", e.target.value)} />
                                <TextField label="Notes" multiline rows={3}
                                           onChange={(e) => handleChange("notes", e.target.value)} />
                            </Stack>
                        )}

                        {bundleType === "BLOOD_GROUP" && (
                            <Stack spacing={2}>
                                <TextField label="ABO" placeholder="A / B / AB / O"
                                           onChange={(e) => handleChange("abo", e.target.value)} />
                                <TextField label="Rh" placeholder="+ / -"
                                           onChange={(e) => handleChange("rh", e.target.value)} />
                                <TextField type="date" label="Last Confirmed"
                                           InputLabelProps={{ shrink: true }}
                                           onChange={(e) => handleChange("last_confirmed", e.target.value)} />
                            </Stack>
                        )}

                        {bundleType === "EMERGENCY_SUMMARY" && (
                            <Stack spacing={2}>
                                <TextField
                                    label="Problems (comma separated)"
                                    onChange={(e) =>
                                        handleChange("problems", e.target.value.split(",").map((s) => s.trim()))
                                    }
                                />
                                <TextField
                                    label="Allergies (comma separated)"
                                    onChange={(e) =>
                                        handleChange("allergies", e.target.value.split(",").map((s) => s.trim()))
                                    }
                                />
                                <TextField
                                    label="Current Medications (comma separated)"
                                    onChange={(e) =>
                                        handleChange("current_medications", e.target.value.split(",").map((s) => s.trim()))
                                    }
                                />
                                <TextField label="Notes" multiline rows={3}
                                           onChange={(e) => handleChange("notes", e.target.value)} />
                            </Stack>
                        )}

                        {/*/!* Submit *!/*/}
                        {/*{bundleType && (*/}
                        {/*    <Button*/}
                        {/*        variant="contained"*/}
                        {/*        sx={{ mt: 4 }}*/}
                        {/*        disabled={loading}*/}
                        {/*        onClick={submit}*/}
                        {/*    >*/}
                        {/*        Save Bundle*/}
                        {/*    </Button>*/}
                        {/*)}*/}

                        <Box display="flex" gap={2}>
                            <Button
                                variant="outlined"
                                onClick={() => setStep("SEARCH")}
                            >
                                Back
                            </Button>

                            <Button
                                variant="contained"
                                disabled={!canSubmit || loading}
                                onClick={handleCreateBundle}
                            >
                                {loading ? <CircularProgress size={20} /> : "Create"}
                            </Button>
                        </Box>
                    </Box>
                </Paper>
            )}

            {/* ================= NOT FOUND MODAL ================= */}
            <Dialog open={openModal} onClose={() => setOpenModal(false)}>
                <DialogTitle>Patient Not Found</DialogTitle>
                <DialogContent>
                    This user is not in our hospital system.
                    Would you like to register this patient?
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setOpenModal(false)}>Cancel</Button>
                    <Button
                        variant="contained"
                        onClick={() => router.push("/doctor/patients/register")}
                    >
                        Register
                    </Button>
                </DialogActions>
            </Dialog>
        </Box>
    );
}