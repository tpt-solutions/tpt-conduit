// White-label theming config. All values are read from NEXT_PUBLIC_* env vars
// so an operator can rebrand the app per-deployment without touching code.
export const theme = {
  brandName: process.env.NEXT_PUBLIC_BRAND_NAME || "TPT Conduit",
  logoUrl: process.env.NEXT_PUBLIC_BRAND_LOGO_URL || "",
  colorPrimary: process.env.NEXT_PUBLIC_COLOR_PRIMARY || "#2451ff",
  colorAccent: process.env.NEXT_PUBLIC_COLOR_ACCENT || "#12b886",
};
