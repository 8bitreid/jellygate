document.addEventListener('DOMContentLoaded', function() {
  const steps = document.querySelectorAll('.tutorial-step');
  const prevBtn = document.getElementById('prevBtn');
  const nextBtn = document.getElementById('nextBtn');
  const skipBtn = document.getElementById('skipBtn');
  const finishBtn = document.getElementById('finishBtn');
  const progressFill = document.getElementById('progressFill');
  const progressText = document.getElementById('progressText');
  const totalSteps = steps.length;
  let currentStep = 1;

  function updateStep() {
    steps.forEach(function(step, index) {
      if (index + 1 === currentStep) {
        step.classList.add('active');
      } else {
        step.classList.remove('active');
      }
    });

    const progress = (currentStep / totalSteps) * 100;
    progressFill.style.width = progress + '%';
    progressText.textContent = currentStep + ' of ' + totalSteps;

    prevBtn.disabled = currentStep === 1;

    if (currentStep === totalSteps) {
      nextBtn.style.display = 'none';
      finishBtn.style.display = 'inline-block';
      skipBtn.style.display = 'none';
    } else {
      nextBtn.style.display = 'inline-block';
      finishBtn.style.display = 'none';
      skipBtn.style.display = 'inline';
    }
  }

  nextBtn.addEventListener('click', function() {
    if (currentStep < totalSteps) {
      currentStep++;
      updateStep();
    }
  });

  prevBtn.addEventListener('click', function() {
    if (currentStep > 1) {
      currentStep--;
      updateStep();
    }
  });

  updateStep();
});
