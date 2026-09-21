import './style.css'

document.addEventListener('DOMContentLoaded', () => {
  const buttons = document.querySelectorAll('.nav-btn');
  const sections = document.querySelectorAll('.doc-section');

  buttons.forEach(btn => {
    btn.addEventListener('click', () => {
      // Remove active class from all buttons
      buttons.forEach(b => b.classList.remove('active'));
      // Add active class to clicked button
      btn.classList.add('active');

      const targetId = btn.getAttribute('data-target');

      // Hide all sections, show target
      sections.forEach(sec => {
        if (sec.id === targetId) {
          sec.classList.remove('hidden');
          // small delay to allow display:block to apply before animating opacity
          setTimeout(() => {
            sec.style.opacity = '1';
            sec.style.transform = 'translateY(0)';
          }, 10);
        } else {
          sec.classList.add('hidden');
          sec.style.opacity = '0';
          sec.style.transform = 'translateY(20px)';
        }
      });
    });
  });
});
