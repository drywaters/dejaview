// Drag and drop functionality for movie reordering within groups
(function() {
    'use strict';

    // Initialize drag and drop for all sortable grids
    function initDragDrop() {
        document.querySelectorAll('.sortable-grid').forEach(initGrid);
    }

    function initGrid(grid) {
        const groupNum = grid.dataset.group;
        if (!groupNum) return;

        let draggedItem = null;
        let placeholder = null;

        grid.querySelectorAll('.draggable-item').forEach(item => {
            item.setAttribute('draggable', 'true');

            item.addEventListener('dragstart', function(e) {
                draggedItem = this;
                this.classList.add('dragging');
                e.dataTransfer.effectAllowed = 'move';
                e.dataTransfer.setData('text/plain', this.dataset.entryId);

                // Create placeholder
                placeholder = document.createElement('div');
                placeholder.className = 'drag-placeholder';
                placeholder.style.height = this.offsetHeight + 'px';
            });

            item.addEventListener('dragend', function(e) {
                this.classList.remove('dragging');
                if (placeholder && placeholder.parentNode) {
                    placeholder.parentNode.removeChild(placeholder);
                }
                placeholder = null;
                draggedItem = null;

                // Save the new order
                saveOrder(grid, groupNum);
            });

            item.addEventListener('dragover', function(e) {
                e.preventDefault();
                e.dataTransfer.dropEffect = 'move';

                if (!draggedItem || this === draggedItem) return;

                const rect = this.getBoundingClientRect();
                const midX = rect.left + rect.width / 2;
                const midY = rect.top + rect.height / 2;

                // Determine if we should insert before or after
                const afterElement = (e.clientX > midX || e.clientY > midY);

                if (afterElement) {
                    this.parentNode.insertBefore(draggedItem, this.nextSibling);
                } else {
                    this.parentNode.insertBefore(draggedItem, this);
                }
            });

            item.addEventListener('drop', function(e) {
                e.preventDefault();
            });
        });

        // Handle drop on the grid itself
        grid.addEventListener('dragover', function(e) {
            e.preventDefault();
        });

        grid.addEventListener('drop', function(e) {
            e.preventDefault();
        });
    }

    function saveOrder(grid, groupNum) {
        const items = grid.querySelectorAll('.draggable-item');
        const entryIds = Array.from(items).map(item => item.dataset.entryId);

        fetch('/api/groups/' + groupNum + '/reorder', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ entry_ids: entryIds })
        })
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to save order');
            }
            // Show success toast via HTMX trigger
            const event = new CustomEvent('showToast', {
                detail: { message: 'Order updated!', type: 'success' }
            });
            document.body.dispatchEvent(event);
        })
        .catch(error => {
            console.error('Error saving order:', error);
            // Show error toast
            const event = new CustomEvent('showToast', {
                detail: { message: 'Failed to save order', type: 'error' }
            });
            document.body.dispatchEvent(event);
        });
    }

    // Initialize on page load
    document.addEventListener('DOMContentLoaded', initDragDrop);

    // Reinitialize after HTMX content swaps
    document.body.addEventListener('htmx:afterSwap', initDragDrop);
    document.body.addEventListener('htmx:afterSettle', initDragDrop);
})();
