"use client";

import {
    Box,
    Button,
    MenuItem,
    Stack,
    TextField,
    Typography,
} from "@mui/material";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import {BundleType} from "@/lib/bundle";

export default function UploadBundlePage() {
    const router = useRouter();
    const [bundleType, setBundleType] = useState<BundleType | "">("");
    const [form, setForm] = useState<any>({});
    const [loading, setLoading] = useState(false);

    const handleChange = (key: string, value: any) => {
        setForm((prev: any) => ({ ...prev, [key]: value }));
    };

    const submit = async () => {
        setLoading(true);
        try {
            await api.post("/healthchain/v1/bundles", {
                type: bundleType,
                data: form,
            });
            router.push("/patient/bundles");
        } catch (e) {
            alert("Failed to upload bundle");
            console.error(e);
        } finally {
            setLoading(false);
        }
    };

    return (
        <Box p={3} maxWidth={700}>
            <Typography variant="h5" fontWeight={600} mb={2} color={"black"}>
                Upload Bundle
            </Typography>

            {/* Select bundle type */}
            <TextField
                select
                fullWidth
                label="Bundle Type"
                value={bundleType}
                onChange={(e) => {
                    setBundleType(e.target.value as BundleType);
                    setForm({});
                }}
                sx={{ mb: 3 }}
            >
                <MenuItem value="ALLERGY">Allergy</MenuItem>
                <MenuItem value="MEDICATION">Medication</MenuItem>
                <MenuItem value="BLOOD_GROUP">Blood Group</MenuItem>
                <MenuItem value="EMERGENCY_SUMMARY">Emergency Summary</MenuItem>
            </TextField>

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

            {/* Submit */}
            {bundleType && (
                <Button
                    variant="contained"
                    sx={{ mt: 4 }}
                    disabled={loading}
                    onClick={submit}
                >
                    Save Bundle
                </Button>
            )}
        </Box>
    );
}