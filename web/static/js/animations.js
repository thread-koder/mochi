import { animate } from "https://cdn.jsdelivr.net/npm/motion@latest/+esm";

// ============================================================================
// Magnetic Cursor Effect
// ============================================================================

const cards = document.querySelectorAll(".magnetic-card");

cards.forEach((card) => {
  let currentAnimation = null;

  card.addEventListener("mousemove", (e) => {
    const rect = card.getBoundingClientRect();
    const centerX = rect.left + rect.width / 2;
    const centerY = rect.top + rect.height / 2;

    const offsetX = (e.clientX - centerX) / (rect.width / 2);
    const offsetY = (e.clientY - centerY) / (rect.height / 2);

    const maxOffset = 15;
    const x = offsetX * maxOffset;
    const y = offsetY * maxOffset;

    if (currentAnimation) currentAnimation.stop();

    currentAnimation = animate(
      card,
      { x, y, scale: 1.02 },
      { type: "spring", stiffness: 300, damping: 30 }
    );
  });

  card.addEventListener("mouseleave", () => {
    if (currentAnimation) currentAnimation.stop();
    currentAnimation = animate(
      card,
      { x: 0, y: 0, scale: 1 },
      { type: "spring", stiffness: 300, damping: 30 }
    );
  });
});
