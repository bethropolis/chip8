/**
 * Svelte action: dispatches a 'click_outside' event when a click occurs outside the given node.
 * @param {HTMLElement} node - The element to detect outside clicks for.
 * @returns {{ destroy(): void }}
 */
export function clickOutside(node) {
  const handleClick = (event) => {
    if (node && !node.contains(event.target) && !event.defaultPrevented) {
      node.dispatchEvent(new CustomEvent("click_outside", node));
    }
  };

  document.addEventListener("click", handleClick, true);

  return {
    destroy() {
      document.removeEventListener("click", handleClick, true);
    },
  };
}
