document.addEventListener('DOMContentLoaded', () => {
    const profileApp = document.getElementById('profile-app');
    const profileError = document.getElementById('profile-error');

    async function fetchProfile() {
        try {
            const response = await fetch('/api/profile');
            if (!response.ok) {
                if (response.status === 404) {
                    profileError.textContent = 'Profile not found. Please create one in settings.';
                    profileApp.style.display = 'none'; // Corrected string literal
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

    fetchProfile();
});
