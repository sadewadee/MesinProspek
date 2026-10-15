// Indonesian text for Kratos messages, by their stable ID (github.com/ory/kratos text/id.go).
// Messages not listed keep Kratos' English text. Copy follows docs/03 §7; a failed login never
// tells whether the email is registered (docs/03 SCR-01).
import type { UiText } from "./kratos";

const TEXTS: Record<number, string> = {
  // Login
  1010001: "Masuk",
  1010002: "Lanjut dengan {provider}",
  1010003: "Konfirmasi bahwa ini memang Anda untuk melanjutkan.",
  1010004: "Selesaikan langkah verifikasi kedua.",
  1010005: "Verifikasi",
  1010006: "Kode autentikasi",
  1010009: "Pakai aplikasi autentikator",
  1010013: "Lanjut",
  1010022: "Masuk",
  // Registration
  1040001: "Daftar",
  1040002: "Lanjut dengan {provider}",
  1040003: "Lanjut",
  1040008: "Kembali",
  // Settings
  1050001: "Perubahan disimpan.",
  1050002: "Tautkan {provider}",
  1050003: "Lepas tautan {provider}",
  1050004: "Lepas aplikasi autentikator",
  1050005: "Kode QR aplikasi autentikator",
  1050017: "Kunci rahasia aplikasi autentikator. Pakai bila kode QR tidak bisa dipindai.",
  // Recovery
  1060001: "Akun berhasil dipulihkan. Ganti kata sandi Anda sekarang.",
  1060003:
    "Bila email ini terdaftar, kode pemulihan sudah kami kirim. Belum menerima email? Periksa ejaan alamatnya.",
  1060007: "Kembali",
  // Field and button labels
  1070001: "Kata sandi",
  1070003: "Simpan",
  1070005: "Kirim",
  1070006: "Verifikasi kode",
  1070007: "Email",
  1070008: "Kirim ulang kode",
  1070009: "Lanjut",
  1070010: "Kode pemulihan",
  1070011: "Kode verifikasi",
  // Verification
  1080001: "Tautan verifikasi sudah dikirim ke email Anda.",
  1080002: "Email Anda sudah terverifikasi.",
  1080003: "Kode verifikasi sudah kami kirim ke email Anda. Belum menerima email? Periksa ejaan alamatnya.",
  // Validation
  4000002: "Wajib diisi.",
  4000003: "Terlalu pendek.",
  4000004: "Formatnya tidak valid.",
  4000005: "Kata sandi ini tidak bisa dipakai. Pilih kata sandi lain.",
  4000006: "Email atau kata sandi salah. Periksa kembali, lalu coba lagi.",
  4000007: "Sudah ada akun dengan email ini. Masuk, atau pulihkan akun bila lupa kata sandi.",
  4000008: "Kode autentikasi salah. Coba lagi.",
  4000010: "Akun belum aktif. Verifikasi email Anda dulu.",
  4000017: "Terlalu panjang.",
  4000028: "Sudah ada akun dengan email ini. Masuk, atau pulihkan akun bila lupa kata sandi.",
  4000031: "Kata sandi terlalu mirip dengan email.",
  4000032: "Kata sandi minimal {min_length} karakter.",
  4000033: "Kata sandi terlalu panjang.",
  4000034: "Kata sandi ini pernah bocor di internet. Pilih kata sandi lain.",
  4000039: "Kata sandi baru harus berbeda dari yang lama.",
  4000040: "Alamat email tidak valid.",
  // Expired or used flows and codes
  4010001: "Waktu formulir masuk habis. Silakan coba lagi.",
  4010008: "Kode tidak valid atau sudah dipakai. Coba lagi.",
  4040001: "Waktu formulir pendaftaran habis. Silakan coba lagi.",
  4050001: "Waktu formulir habis. Silakan coba lagi.",
  4060004: "Tautan pemulihan tidak valid atau sudah dipakai.",
  4060005: "Waktu pemulihan habis. Silakan coba lagi.",
  4060006: "Kode pemulihan tidak valid atau sudah dipakai. Coba lagi.",
  4070001: "Tautan verifikasi tidak valid atau sudah dipakai.",
  4070005: "Waktu verifikasi habis. Silakan coba lagi.",
  4070006: "Kode verifikasi tidak valid atau sudah dipakai. Coba lagi.",
};

/** Indonesian text of a Kratos message, or its English text when there is none. */
export function textOf(message: UiText): string {
  const template = TEXTS[message.id];
  if (!template) return message.text;
  return template.replace(/\{(\w+)\}/g, (_, key: string) => {
    const value = String(message.context?.[key] ?? "");
    return key === "provider" ? value.charAt(0).toUpperCase() + value.slice(1) : value;
  });
}
