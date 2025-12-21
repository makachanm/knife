document.addEventListener('DOMContentLoaded', () => {
    const profileApp = document.getElementById('profile-app');
    const profileError = document.getElementById('profile-error');
    const recentPostsContainer = document.getElementById('recent-posts-container');

    async function fetchProfile() {
        try {
            const response = await fetch('/api/profile');
            if (!response.ok) {
                if (response.status === 404) {
                    profileError.textContent = 'Profile not found. Please create one in settings.';
                    profileApp.style.display = 'none';
                } else {
                    throw new Error('Could not fetch profile');
                }
                return;
            }
            const profile = await response.json();
            renderProfile(profile);
        } catch (error) {
            profileError.textContent = `Error fetching profile: ${error.message}`;
            console.error('Failed to fetch profile:', error);
        }
    }

    function renderProfile(profile) {
        document.getElementById('profile-name-header').textContent = profile.display_name || 'Your Name';
        document.getElementById('profile-finger').textContent = profile.finger || '@yourhandle';
        document.getElementById('profile-bio').textContent = profile.bio || 'No bio provided.';
    }

    async function fetchNotes() {
        try {
            const response = await fetch('/api/notes');
            if (!response.ok) {
                throw new Error('Could not fetch notes');
            }
            const notes = await response.json();
            renderNotes(notes);
        } catch (error) {
            recentPostsContainer.innerHTML = `<p class='error-message'>Error fetching timeline: ${error.message}</p>`;
            console.error('Failed to fetch notes:', error);
        }
    }

    function renderNotes(notes) {
        if (!notes || notes.length === 0) {
            recentPostsContainer.innerHTML = '<p>No notes yet.</p>';
            return;
        }

        recentPostsContainer.innerHTML = '';
        notes.forEach(note => {
            const noteElement = document.createElement('div');
            noteElement.className = 'note';
            noteElement.dataset.noteId = note.id;

            const createTime = new Date(note.create_time).toLocaleString();

            noteElement.innerHTML = `
                <div class='note-header'>
                    <div>
                        <span class='author'>${escapeHTML(note.author_name)}</span>
                        <span class='finger'>@${escapeHTML(note.author_finger)}</span>
                    </div>
                </div>
                <div class='note-content'></div>
                <div class='note-meta'>
                    <a href='/notes/${note.id}' class='note-link-time'>Posted on ${createTime}</a>
                </div>
            `;

            // Insert HTML content into the note-content div
            const noteContentDiv = noteElement.querySelector('.note-content');
            noteContentDiv.innerHTML = note.content; // Directly insert HTML content

            recentPostsContainer.appendChild(noteElement);
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

    fetchProfile();
    fetchNotes();
});