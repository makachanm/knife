document.addEventListener('DOMContentLoaded', () => {
    const profileForm = document.getElementById('profile-form');
    const formMessage = document.getElementById('form-message');
    const nameInput = document.getElementById('name');
    const bioTextarea = document.getElementById('bio');

    // Fetch current profile data to pre-fill the form
    async function loadProfileForEdit() {
        try {
            const response = await fetch('/api/profile');
            if (!response.ok) {
                // If profile not found, it means it's a new user,
                // and they are effectively creating their profile.
                if (response.status === 404) {
                    formMessage.textContent = 'No profile found. Please create one.';
                    return;
                }
                throw new Error('Could not fetch profile for editing');
            }
            const profile = await response.json();
            nameInput.value = profile.display_name || '';
            bioTextarea.value = profile.bio || '';
        } catch (error) {
            formMessage.textContent = `Error loading profile: ${error.message}`;
            console.error('Failed to load profile for edit:', error);
        }
    }

    // Handle form submission for updating or creating a profile
    profileForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        formMessage.textContent = '';

        const name = nameInput.value.trim();
        const bio = bioTextarea.value.trim();

        if (!name) {
            formMessage.textContent = 'Name is required.';
            return;
        }

        const profileData = {
            display_name: name,
            bio: bio,
        };

        try {
            // First, try to get the profile. If it exists, we PUT (update).
            // If it doesn't exist (404), we POST (create).
            const getResponse = await fetch('/api/profile');
            let method = 'POST';
            let url = '/api/profile';

            if (getResponse.ok) {
                // Profile exists, so we will update it.
                method = 'PUT';
            } else if (getResponse.status !== 404) {
                // Some other error occurred when trying to get the profile.
                const errorData = await getResponse.json();
                throw new Error(errorData.description || 'Failed to check profile existence.');
            }

            const response = await fetch(url, {
                method: method,
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(profileData),
            });

            if (!response.ok) {
                 const errorData = await response.json();
                 throw new Error(errorData.description || 'Failed to save profile');
            }
            
            formMessage.style.color = 'green';
            formMessage.textContent = 'Profile saved successfully!';

        } catch (error) {
            formMessage.style.color = 'red';
            formMessage.textContent = `Error: ${error.message}`;
            console.error('Failed to save profile:', error);
        }
    });

    loadProfileForEdit();
});
