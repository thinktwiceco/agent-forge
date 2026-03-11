id: navigate-gmail
name: Navigate to Gmail Login Page

description: |
  Navigate to Gmail login page and verify the email input field is present.
  
  From the gmail_login.steps.md document:
  "Action: Navigate to `https://gmail.com`"
  
  Expected Result:
  - Page redirects to Google Sign-in: `accounts.google.com/v3/signin/identifier`
  - Page title: "Gmail" or "Sign in - Google Accounts"
  - Email input field is visible

tools:
  - web_browser

instruction: |
  ## Navigate to Gmail
  
  1. **Navigate** to `https://gmail.com`
  
  2. **Verify** the page loaded correctly:
     - Page URL should contain `accounts.google.com/v3/signin`
     - Page title should be "Sign in" or "Gmail"
  
  3. **Check** for the email input field presence:
     - Look for selector: `input[type="email"]` or `#identifierId`
  
  4. **Confirm** the "Next" button is visible
     - Look for selector: `button[data-primary="true"]` or `#identifierNext`

expected_result: |
  Page displays Google Sign-in with email input field visible and ready for input.

on_success:
  - next: |END
