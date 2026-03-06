id: enter-password
title: Enter Password and Complete Login

description: |
  Fill the password input field using vault credentials and submit. 
  
  CRITICAL: Use ONLY `#passwordNext` selector for password submit button.
  The `button[data-primary="true"]` selector TIMES OUT on Google's password page.
  
  From the gmail_login.steps.md document:
  "Selector (CONFIRMED WORKING): `#passwordNext`"
  "⚠️ Important Note: `#passwordNext` ID-based selector instead"
  "Vault Key: `gmail-password`"

prerequisites:
  - vault_key: gmail-password

tools:
  - web_browser
  - vault

instruction: |
  ## Enter Password
  
  1. **Wait** for password page to fully load:
     - Verify URL: `accounts.google.com/v3/signin/challenge/pwd`
     - Verify email address displayed matches expected account
  
  2. **Fill** the password field using vault secret `gmail-password`:
     - Selector: `input[type="password"]` or `#password`
     - Use action: `fill_secret`
  
  3. **Wait** briefly (500ms) for field population
  
  4. **CRITICAL - Use correct selector:**
     - ✅ Click `#passwordNext` (this ID works)
     - ❌ DO NOT use `button[data-primary="true"]` (this TIMES OUT)
  
  5. **Submit** using ONLY this selector:
     ```
     selector: "#passwordNext"
     ```

expected_result: |
  Page redirects to Gmail inbox or shows security challenge if additional verification required.

on_failure:
  - condition: "Context deadline exceeded" or timeout
    remedy: |
      CRITICAL: You used `button[data-primary="true"]` or `[data-primary]` attribute.
      MUST use: `#passwordNext` ID selector instead.
      The attribute selectors timeout on Google's password page.
  
  - condition: "Wrong password" error displayed
    remedy: |
      - Verify vault key `gmail-password` is current
      - Check if account has 2FA enabled (requires additional steps)
      - Google may detected "unusual activity" - check for CAPTCHA
  
  - condition: Password field not found
    remedy: |
      - Wait longer for page transition (increase settle_ms to 1000-2000ms)
      - Try alternative selector: `#password` instead of `input[type="password"]`
  
  - condition: CAPTCHA or 2FA prompt appears
    remedy: |
      This is expected for accounts with additional security.
      Log in via browser normally once to authorize this session.

on_success:
  - next: verify-inbox
