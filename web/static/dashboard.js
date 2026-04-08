// Copy invite URL to clipboard
function copyToClipboard(url, button) {
  // Store original content
  const originalHTML = button.innerHTML;
  
  // Try modern Clipboard API first
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(url).then(() => {
      showCopySuccess(button, originalHTML);
    }).catch(err => {
      console.error('Clipboard API failed:', err);
      fallbackCopy(url, button, originalHTML);
    });
  } else {
    // Fallback for non-secure contexts or older browsers
    fallbackCopy(url, button, originalHTML);
  }
}

// Fallback copy method using textarea
function fallbackCopy(url, button, originalHTML) {
  const textarea = document.createElement('textarea');
  textarea.value = url;
  textarea.style.position = 'fixed';
  textarea.style.left = '-9999px';
  textarea.style.top = '0';
  document.body.appendChild(textarea);
  
  try {
    textarea.select();
    textarea.setSelectionRange(0, 99999); // For mobile devices
    const successful = document.execCommand('copy');
    
    if (successful) {
      showCopySuccess(button, originalHTML);
    } else {
      alert('Failed to copy URL. Please copy manually:\n' + url);
    }
  } catch (err) {
    console.error('Fallback copy failed:', err);
    alert('Failed to copy URL. Please copy manually:\n' + url);
  } finally {
    document.body.removeChild(textarea);
  }
}

// Show success state
function showCopySuccess(button, originalHTML) {
  button.classList.add('copied');
  button.innerHTML = `
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
    </svg>
    copied!
  `;
  
  // Reset after 2 seconds
  setTimeout(() => {
    button.classList.remove('copied');
    button.innerHTML = originalHTML;
  }, 2000);
}

// Format a Date as "YYYY-MM-DDTHH:MM" in local time (for datetime-local input)
function toLocalDateTimeString(date) {
  const pad = n => String(n).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth()+1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

// Format a Date as "YYYY-MM-DDTHH:MM" in UTC (for server submission)
function toUTCDateTimeString(date) {
  const pad = n => String(n).padStart(2, '0');
  return `${date.getUTCFullYear()}-${pad(date.getUTCMonth()+1)}-${pad(date.getUTCDate())}T${pad(date.getUTCHours())}:${pad(date.getUTCMinutes())}`;
}

// Initialize all copy buttons
document.addEventListener('DOMContentLoaded', () => {
  const copyButtons = document.querySelectorAll('.btn-copy');
  copyButtons.forEach(button => {
    button.addEventListener('click', (e) => {
      e.preventDefault();
      const url = button.getAttribute('data-url');
      copyToClipboard(url, button);
    });
  });

  // Default expires_at to 7 days from now (shown in user's local time)
  const expiresInput = document.getElementById('expires_at');
  if (expiresInput && !expiresInput.value) {
    const sevenDays = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000);
    expiresInput.value = toLocalDateTimeString(sevenDays);
  }

  // On submit, convert expires_at from local time to UTC before sending to server
  const inviteForm = expiresInput && expiresInput.closest('form');
  if (inviteForm) {
    inviteForm.addEventListener('submit', () => {
      if (expiresInput.value) {
        // new Date("YYYY-MM-DDTHH:MM") is treated as local time by browsers
        const local = new Date(expiresInput.value);
        expiresInput.value = toUTCDateTimeString(local);
      }
    });
  }
});
