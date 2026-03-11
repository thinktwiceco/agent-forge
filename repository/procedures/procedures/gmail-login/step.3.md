id: verify-inbox
name: Verify Gmail Inbox Access

description: |
  Confirm successful authentication by verifying Gmail inbox loaded with all expected elements visible.
  
  Success Criteria from gmail_login.steps.md:
  - URL contains `mail.google.com`
  - Page displays Gmail inbox with email list
  - "Compose" button visible
  - No authentication prompts remaining

tools:
  - web_browser

instruction: |
  ## Verify Login Success
  
  After password submission, verify the following success indicators:
  
  1. **Check URL**
     - Should contain: `mail.google.com`
     - NOT `accounts.google.com` (would indicate still on login page)
  
  2. **Verify Gmail Interface Elements:**
     - Look for "Compose" button
     - Look for email list/containers
     - Look for Gmail navigation sidebar
  
  3. **Confirm No Auth Prompts:**
     - No "Wrong password" message
     - No "Sign in" buttons
     - No CAPTCHA (if present, additional steps needed)
  
  ## Success Criteria (ALL must be true):
  ✅ URL contains `mail.google.com`
  ✅ Page displays Gmail inbox with email list
  ✅ "Compose" button visible
  ✅ No authentication prompts remaining
  
  If all criteria met, login is COMPLETE.

expected_result: |
  Gmail inbox fully loaded with email list visible and compose functionality available.

on_failure:
  - condition: Still on `accounts.google.com` URL
    remedy: |
      - Check for error messages (Wrong password, etc.)
      - Go back to previous step and retry
      - Check vault credentials
  
  - condition: Security challenge/CAPTCHA appears
    remedy: |
      Account requires additional verification:
      - Log in manually via browser to authorize this device
      - Check for 2FA prompts
      - May need to solve CAPTCHA manually once
  
  - condition: Empty inbox or missing elements
    remedy: |
      - Wait longer for page to fully load (increase settle_ms)
      - Refresh page to trigger complete render

on_success:
  - action: complete
    output: |
      ✅ Gmail login complete!
      - URL: {current_url}
      - User authenticated successfully
      - Inbox accessible

on_failure:
  - action: fail
    reason: |
      Could not verify Gmail inbox access. Review previous steps and vault credentials.
