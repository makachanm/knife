document.addEventListener('DOMContentLoaded', () => {
    const timeline = document.getElementById('timeline');
    const categoryTitle = document.getElementById('category-title');
    
    // Extract category name from URL path /category/{name}
    const pathParts = window.location.pathname.split('/');
    // Assuming /category/name, so name is at index 2 (0='', 1='category', 2='name')
    const categoryName = decodeURIComponent(pathParts[2] || '');

    if (!categoryName) {
        categoryTitle.textContent = "Category not found";
        timeline.innerHTML = '<p>No category specified.</p>';
        return;
    }

    categoryTitle.textContent = `Category: ${categoryName}`;

    fetchCategoryNotes(categoryName);

    async function fetchCategoryNotes(name) {
        try {
            const response = await fetch(`/api/category/${encodeURIComponent(name)}`);
            if (!response.ok) {
                throw new Error('Could not fetch notes for this category');
            }
            const notes = await response.json();
            renderNotes(notes);
        } catch (error) {
            timeline.innerHTML = `<p class='error-message'>Error fetching timeline: ${error.message}</p>`;
            console.error('Failed to fetch notes:', error);
        }
    }

    function renderNotes(notes) {
        if (!notes || notes.length === 0) {
            timeline.innerHTML = '<p>No notes found in this category.</p>';
            return;
        }

        timeline.innerHTML = '';
        notes.forEach(note => {
            const noteElement = NoteRenderer.createNoteElement(note);
            timeline.appendChild(noteElement);
        });
    }

    function escapeHTML(str) {
        if (str === null || str === undefined) {
            return '';
        }
        return str.toString()
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#039;');
    }
});
