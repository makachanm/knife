document.addEventListener('DOMContentLoaded', () => {
    const noteForm = document.getElementById('note-form');
    const formError = document.getElementById('form-error');

    noteForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        formError.textContent = '';

        const formData = new FormData(noteForm);
        const content = formData.get('content');
        const cw = formData.get('cw');
        const publicRange = parseInt(formData.get('public_range'), 10);

        if (!content) {
            formError.textContent = 'Content is required.';
            return;
        }

        const note = {
            content: content,
            cw: cw,
            public_range: publicRange,
        };

        try {
            const response = await fetch('/api/notes', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(note),
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.description || 'Failed to create note');
            }

            window.location.href = '/';
        } catch (error) {
            formError.textContent = `Error: ${error.message}`;
            console.error('Failed to create note:', error);
        }
    });
});
