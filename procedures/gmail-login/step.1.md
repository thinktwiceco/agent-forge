id: enter-email
title: Enter Gmail Email Address

description: |
  Fill the email input field using credentials stored in vault and submit to proceed to password entry.
  
  From the gmail_login.steps.md document:
  "Selector: `input[type="email"]` or `#identifierId`"
  "Vault Key: `gmail-username`"
  
  Action: Fill the email field using vault credentials
  Expected Result: Email field populated with stored username

prerequisites:
  - vault_key: gmail-username
  - vault_key: gmail-password

tools:
  - web_browser
  - vault

instruction: |
  ## Enter Email Address
  
  1. **Wait** for email input field to be visible:
     - Selector: `input[type="email"]` or `#identifierId`
  
  2. **Fill** the email field using vault secret `gmail-username`:
     - Use action: `fill_secret`
     - Element: `input[type="email"]` or `#identifierId`
  
  3. **Wait** briefly (500ms) for field population
  
  4. **Click** the "Next" button to submit:
     - Selector: `button[data-primary="true"]` or `#identifierNext`
  
  5. **Verify** page transition:
     - URL changes to: `accounts.google.com/v3/signin/challenge/pwd`
     - Password input field appears: `input[type="password"]` or `#password`
     - Confirmed email address displayed on page

expected_result: |
  Page transitions to password entry screen with email address confirmed and password field visible.

on_failure:
  - condition: Timeout waiting for email field
    remedy: |
      - Refresh page and retry
      - Check if network connection is stable
      - Try navigating directly to `https://accounts.google.com/signin`
  
  - condition: "Couldn't find element"
    remedy: |
      - Try alternative selector: `#identifierId` instead of `input[type="email"]`
      - Wait longer for page load (increase settle_ms)
  
  - condition: Page doesn't transition after clicking Next
    remedy: |
      - Verify vault key `gmail-username` contains valid email
      - Check for CAPTCHA or security challenge
      - Screenshot page to diagnose issue

on_success:
  - next: enter-password
