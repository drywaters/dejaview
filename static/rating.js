// Rating input focus management
// Preserves focus when tabbing between rating inputs during HTMX swaps
(function() {
    'use strict';

    let pendingFocusPersonId = null;
    let lastFocusedPersonId = null;

    function isRatingInput(el) {
        return el &&
            el.tagName === 'INPUT' &&
            el.name === 'score' &&
            el.dataset.personId;
    }

    document.body.addEventListener('focusin', function(evt) {
        const target = evt.target;
        if (isRatingInput(target)) {
            lastFocusedPersonId = target.dataset.personId;
            return;
        }

        if (target && target.closest && !target.closest('#ratings-section')) {
            lastFocusedPersonId = null;
        }
    });

    document.body.addEventListener('pointerdown', function(evt) {
        const target = evt.target;
        if (target && target.closest && !target.closest('#ratings-section')) {
            lastFocusedPersonId = null;
        }
    });

    // Before HTMX swaps the DOM, capture the currently focused rating input
    // This fires AFTER the user has tabbed to the next input
    document.body.addEventListener('htmx:beforeSwap', function(evt) {
        // Only handle swaps targeting elements within the ratings section
        const target = evt.detail.target;
        if (!target) return;

        const ratingsSection = target.id === 'ratings-section'
            ? target
            : (target.closest ? target.closest('#ratings-section') : null);
        if (!ratingsSection) return;

        // Check if a rating input is currently focused
        const activeEl = document.activeElement;
        if (isRatingInput(activeEl)) {
            pendingFocusPersonId = activeEl.dataset.personId;
            return;
        }

        pendingFocusPersonId = lastFocusedPersonId;
    });

    // After HTMX settles, restore focus to the rating input
    document.body.addEventListener('htmx:afterSettle', function(evt) {
        if (!pendingFocusPersonId) return;

        const personId = pendingFocusPersonId;
        pendingFocusPersonId = null;

        // Use requestAnimationFrame to ensure DOM is fully rendered
        requestAnimationFrame(function() {
            const input = document.querySelector(
                '#ratings-section input[data-person-id="' + personId + '"]'
            );
            if (input) {
                input.focus();
            }
        });
    });
})();
