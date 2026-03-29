// app/lib/qr.ts
export type DoctorQRPayload = {
    type: "DOCTOR_QR";
    doctorHash: string;
    issuedAt?: number;
    nonce?: string;
};

/**
 * Expected QR formats:
 *  - "HC:<base64-json>"
 *  - "<base64-json>" (fallback)
 */
export function decodeDoctorQR(raw: string): { doctorHash: string } {
    const trimmed = (raw || "").trim();
    if (!trimmed) throw new Error("QR value is empty");

    const base64 = trimmed.startsWith("HC:") ? trimmed.slice(3) : trimmed;

    let decoded: DoctorQRPayload;
    try {
        decoded = JSON.parse(atob(base64));
    } catch {
        throw new Error("Invalid QR payload (cannot decode base64/json)");
    }

    if (decoded.type !== "DOCTOR_QR") throw new Error("Invalid QR type");
    if (!decoded.doctorHash) throw new Error("QR missing doctorHash");

    return { doctorHash: decoded.doctorHash };
}