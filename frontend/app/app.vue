<template>
  <div>
    <NuxtPage />
  </div>
</template>

<style>
/* Global design tokens and the category badge system — shared by every
   page (pages/index.vue's product cards, pages/rates.vue's Type/Category
   column). Page-specific chrome (hero, tabs, table layout, etc.) lives in
   each page's own scoped <style> block.

   Theme is explicit (composables/useTheme.ts sets <html data-theme="...">
   and persists the choice), not tied to the OS's prefers-color-scheme —
   :root below is the light theme and is always the default; the
   [data-theme="dark"] block only applies once the user actually toggles
   to dark (or a stored preference re-applies it on return). */
:root {
  --bg: #f7f7f8;
  --panel: #ffffff;
  --border: #e2e2e6;
  --text: #1c1c1f;
  --muted: #6b6b74;
  --accent: #2f6fed;
  --navy: #1a2b6b;
  --navy-dark: #0f1830;
  --footer-bg: #12172b;
  --footer-text: #c7cadb;
  --footer-muted: #7d84a3;
}
:root[data-theme='dark'] {
  --bg: #121214;
  --panel: #1b1c1f;
  --border: #303136;
  --text: #eceef0;
  --muted: #9a9aa2;
  --accent: #6c9bff;
  --navy: #14225c;
  --navy-dark: #0a1226;
  --footer-bg: #0a0d1a;
  --footer-text: #c7cadb;
  --footer-muted: #6a7092;
}

* {
  box-sizing: border-box;
}
body {
  margin: 0;
  background: var(--bg);
  color: var(--text);
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
}
a {
  color: inherit;
}

.badge {
  display: inline-block;
  padding: 0.15rem 0.5rem;
  border-radius: 999px;
  font-size: 0.72rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.02em;
  /* Fallback for any category code without a specific rule below, so a
     new taxonomy code never renders unstyled. */
  background: var(--border);
  color: var(--text);
}
/* Fixed deposit categories */
.badge-standard_fd { background: #e6f0ff; color: #1a4fb4; }
.badge-senior_citizen_fd { background: #fdeedd; color: #a35a10; }
.badge-special_fd { background: #ecebff; color: #4a3fd6; }
.badge-sathkara_fd { background: #fde8ef; color: #b3245c; }
.badge-digital_fd { background: #e4f8ef; color: #157a4d; }
/* Savings categories */
.badge-standard_savings { background: #e0f7fa; color: #00707a; }
.badge-senior_savings { background: #fff4d6; color: #8a6100; }
.badge-teen_savings { background: #e3f2fd; color: #0d5f9e; }
.badge-womens_savings { background: #fde4f0; color: #a01764; }
.badge-minor_savings { background: #f1f8e0; color: #5c7a10; }
/* Loan categories */
.badge-housing_loan { background: #e8e6fd; color: #3f2fb0; }
.badge-personal_loan { background: #e1f0ff; color: #1660b0; }
.badge-education_loan { background: #f2e6fd; color: #6a1fb0; }
.badge-gold_loan { background: #fff6da; color: #8a6a00; }
.badge-lease { background: #eef1f5; color: #3d4a5c; }
.badge-pensioner_loan { background: #fbe4e6; color: #9c1f2e; }
[data-theme='dark'] .badge-standard_fd { background: #17335f; color: #a8c6ff; }
[data-theme='dark'] .badge-senior_citizen_fd { background: #4a331a; color: #ffcf9e; }
[data-theme='dark'] .badge-special_fd { background: #2c2867; color: #cac6ff; }
[data-theme='dark'] .badge-sathkara_fd { background: #5b1836; color: #ffb3cf; }
[data-theme='dark'] .badge-digital_fd { background: #103a26; color: #96f0c4; }
[data-theme='dark'] .badge-standard_savings { background: #0d3a3e; color: #7fe0eb; }
[data-theme='dark'] .badge-senior_savings { background: #4a3a0a; color: #ffd980; }
[data-theme='dark'] .badge-teen_savings { background: #123a5c; color: #8fcbff; }
[data-theme='dark'] .badge-womens_savings { background: #4a1030; color: #ff9ecb; }
[data-theme='dark'] .badge-minor_savings { background: #2e3a10; color: #c8e88a; }
[data-theme='dark'] .badge-housing_loan { background: #241a5c; color: #bcb2ff; }
[data-theme='dark'] .badge-personal_loan { background: #0f2e57; color: #9cc6ff; }
[data-theme='dark'] .badge-education_loan { background: #341a57; color: #d3aeff; }
[data-theme='dark'] .badge-gold_loan { background: #453400; color: #ffd76b; }
[data-theme='dark'] .badge-lease { background: #232a33; color: #c2cdd9; }
[data-theme='dark'] .badge-pensioner_loan { background: #4a1116; color: #ff9aa3; }
</style>
