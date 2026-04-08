document.addEventListener('DOMContentLoaded', function() {
  const form = document.querySelector('form');
  const password = document.getElementById('password');
  const confirm = document.getElementById('confirm');
  const matchMsg = document.getElementById('password-match-msg');
  
  // Password strength requirements
  const requirements = {
    length: { regex: /.{8,}/, element: document.getElementById('req-length'), check: document.getElementById('check-length') },
    number: { regex: /\d/, element: document.getElementById('req-number'), check: document.getElementById('check-number') },
    special: { regex: /[!@#$%^&*(),.?":{}|<>]/, element: document.getElementById('req-special'), check: document.getElementById('check-special') }
  };
  
  function validatePasswordStrength() {
    if (!password) return false;
    
    let allValid = true;
    const pwd = password.value;
    
    Object.keys(requirements).forEach(key => {
      const req = requirements[key];
      const isValid = req.regex.test(pwd);
      
      if (req.element && req.check) {
        if (isValid) {
          req.check.textContent = '✓';
          req.element.style.color = '#16a34a';
        } else {
          req.check.textContent = '○';
          req.element.style.color = '#666';
          allValid = false;
        }
      }
    });
    
    return allValid;
  }
  
  function validatePasswords() {
    if (!password || !confirm) return false;
    
    if (confirm.value === '') {
      if (matchMsg) {
        matchMsg.style.display = 'none';
      }
      return false;
    }
    
    if (password.value !== confirm.value) {
      if (matchMsg) {
        matchMsg.textContent = '❌ Passwords do not match';
        matchMsg.style.color = '#dc2626';
        matchMsg.style.display = 'block';
      }
      return false;
    } else {
      if (matchMsg) {
        matchMsg.textContent = '✓ Passwords match';
        matchMsg.style.color = '#16a34a';
        matchMsg.style.display = 'block';
      }
      return true;
    }
  }
  
  function updateSubmitButton() {
    const strengthValid = validatePasswordStrength();
    const matchValid = validatePasswords();
    const username = document.getElementById('username');
    const usernameValid = username && username.value.length > 0;
    const submitBtn = form?.querySelector('button[type="submit"]');
    
    if (submitBtn) {
      if (strengthValid && matchValid && usernameValid) {
        submitBtn.disabled = false;
        submitBtn.style.opacity = '1';
        submitBtn.style.cursor = 'pointer';
      } else {
        submitBtn.disabled = true;
        submitBtn.style.opacity = '0.5';
        submitBtn.style.cursor = 'not-allowed';
      }
    }
  }
  
  if (password) {
    password.addEventListener('input', updateSubmitButton);
  }
  
  if (confirm) {
    confirm.addEventListener('input', updateSubmitButton);
  }
  
  const username = document.getElementById('username');
  if (username) {
    username.addEventListener('input', updateSubmitButton);
  }
  
  // Initialize submit button state
  updateSubmitButton();
  
  form?.addEventListener('submit', function(e) {
    const strengthValid = validatePasswordStrength();
    const matchValid = validatePasswords();
    
    if (!strengthValid || !matchValid) {
      e.preventDefault();
    }
  });
});
