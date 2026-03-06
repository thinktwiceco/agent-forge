# Gmail Login Algorithm

## Goal
Successfully authenticate into a Gmail account and reach the inbox.

## Prerequisites
- Valid Gmail credentials stored in vault:
  - `gmail-username`: Email address
  - `gmail-password`: Account password
- Web browser access

## Algorithm

### Step 1: Navigate to Gmail Login Page
**Action:** Navigate to `https://gmail.com`

**Expected Result:** 
- Page redirects to Google Sign-in: `accounts.google.com/v3/signin/identifier`
- Page title: "Gmail" or "Sign in - Google Accounts"
- Email input field is visible

**Observation Checklist:**
- [ ] Page loads without errors
- [ ] Email input field present (`input[type="email"]` or `#identifierId`)
- [ ] "Next" button visible

---

### Step 2: Enter Email Address
**Action:** Fill the email field using vault credentials

**Selector:** `input[type="email"]` or `#identifierId`

**Vault Key:** `gmail-username`

**Expected Result:**
- Email field populated with stored username
- No errors displayed

---

### Step 3: Submit Email
**Action:** Click the "Next" button to proceed to password page

**Selector:** `button[data-primary="true"]` or `#identifierNext`

**Expected Result:**
- Page transitions to password entry screen
- URL changes to: `accounts.google.com/v3/signin/challenge/pwd`
- Password input field is visible
- Email address displayed as confirmation

**Observation Checklist:**
- [ ] Password input field present (`input[type="password"]`)
- [ ] Confirmed email visible on page
- [ ] "Next" button visible

---

### Step 4: Enter Password
**Action:** Fill the password field using vault credentials

**Selector:** `input[type="password"]` or `#password`

**Vault Key:** `gmail-password`

**Expected Result:**
- Password field populated (characters hidden)
- No errors displayed

---

### Step 5: Submit Password
**Action:** Click the "Next" button to complete authentication

**Selector (CONFIRMED WORKING):** `#passwordNext`

**⚠️ Important Note:** 
- `button[data-primary="true"]` or `[data-primary]` selectors **TIME OUT** on Google's password page
- Use `#passwordNext` ID-based selector instead

**Expected Results:**

**SUCCESS:**
- Redirect to Gmail inbox (`mail.google.com`)
- Page shows Gmail interface with emails, compose button, etc.
- Login complete ✅

**FAILURE (Credential Issue):**
- Page shows: "Wrong password. Try again or click 'Forgot password?'"
- Remains on password entry page
- Requires password verification or account recovery

**FAILURE (Security Challenge):**
- CAPTCHA or 2FA prompt appears
- Additional verification required

---

## Troubleshooting Guide

### "Context deadline exceeded" Error
**Problem:** Button click times out
**Solution:** 
- Use ID-based selectors (`#passwordNext`) instead of attribute-based (`[data-primary]`)
- Wait for page stability before clicking

### "Wrong password" Error
**Problem:** Stored password is incorrect or account requires additional verification
**Solution:**
- Verify credentials in vault are current
- Check if account has 2FA enabled (requires additional steps)
- Check if Google detected "unusual activity" and requires CAPTCHA

### Page Not Loading
**Problem:** Network or navigation error
**Solution:**
- Try navigating directly to `https://accounts.google.com/signin`
- Clear browser session and retry

---

## Summary of Working Selectors

| Step | Element | Working Selector | Problematic Selector |
|------|---------|------------------|---------------------|
| 1 | Email Input | `input[type="email"]`, `#identifierId` | — |
| 2 | Email Next | `button[data-primary="true"]`, `#identifierNext` | — |
| 3 | Password Input | `input[type="password"]`, `#password` | — |
| 4 | Password Next | `#passwordNext` | `button[data-primary="true"]` (times out) |

---

## Success Criteria
✅ **Goal Achieved When:**
- URL contains `mail.google.com`
- Page displays Gmail inbox with email list
- "Compose" button visible
- No authentication prompts remaining

---

*Document Version: 1.0*
*Created: 2026-03-06*
*Tested With: Google Accounts v3 Sign-in Flow*
