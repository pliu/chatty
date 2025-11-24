let currentUser = null;
let currentUserID = null;
let currentChat = null;
let ws = null;
// Check for existing session
// We do NOT auto-login because we need the password to decrypt the private key.
// If the page is refreshed, the memory is cleared, so the user must log in again.
document.getElementById('auth-section').style.display = 'flex';
document.getElementById('chat-section').style.display = 'none';
showTab('login');

// Auth
function showTab(tab) {
    document.getElementById('login-form').style.display = tab === 'login' ? 'flex' : 'none';
    document.getElementById('signup-form').style.display = tab === 'signup' ? 'flex' : 'none';
}

async function hashPassword(password) {
    const msgBuffer = new TextEncoder().encode(password);
    const hashBuffer = await crypto.subtle.digest('SHA-256', msgBuffer);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
}

let currentChatID = null;
let chatKeys = {}; // Map chatID -> decrypted symmetric key
let sessionPrivateKey = null; // CryptoKey object for ECDH

// Helper functions for base64 encoding/decoding
function toBase64(buffer) {
    return btoa(String.fromCharCode(...new Uint8Array(buffer)));
}

function fromBase64(base64) {
    const binary = atob(base64);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
        bytes[i] = binary.charCodeAt(i);
    }
    return bytes;
}

// Generate ECDH key pair for asymmetric encryption
async function generateKeyPair() {
    return await crypto.subtle.generateKey(
        {
            name: "ECDH",
            namedCurve: "P-256"
        },
        true, // extractable
        ["deriveKey", "deriveBits"]
    );
}

// Derive encryption key from password using PBKDF2
async function deriveKeyFromPassword(password, salt) {
    const passwordKey = await crypto.subtle.importKey(
        "raw",
        new TextEncoder().encode(password),
        "PBKDF2",
        false,
        ["deriveKey"]
    );

    return await crypto.subtle.deriveKey(
        {
            name: "PBKDF2",
            salt: salt,
            iterations: 100000,
            hash: "SHA-256"
        },
        passwordKey,
        { name: "AES-GCM", length: 256 },
        false,
        ["encrypt", "decrypt"]
    );
}

// Encrypt private key with password
async function encryptPrivateKey(privateKey, password) {
    // Export the private key to raw format
    const privateKeyRaw = await crypto.subtle.exportKey("jwk", privateKey);
    const privateKeyBytes = new TextEncoder().encode(JSON.stringify(privateKeyRaw));

    // Generate salt and IV
    const salt = crypto.getRandomValues(new Uint8Array(16));
    const iv = crypto.getRandomValues(new Uint8Array(12));

    // Derive encryption key from password
    const encryptionKey = await deriveKeyFromPassword(password, salt);

    // Encrypt the private key
    const encrypted = await crypto.subtle.encrypt(
        { name: "AES-GCM", iv: iv },
        encryptionKey,
        privateKeyBytes
    );

    // Combine salt + IV + ciphertext
    const combined = new Uint8Array(salt.length + iv.length + encrypted.byteLength);
    combined.set(salt, 0);
    combined.set(iv, salt.length);
    combined.set(new Uint8Array(encrypted), salt.length + iv.length);

    return toBase64(combined);
}

// Decrypt private key with password
async function decryptPrivateKey(encryptedData, password) {
    const combined = fromBase64(encryptedData);

    // Extract salt, IV, and ciphertext
    const salt = combined.slice(0, 16);
    const iv = combined.slice(16, 28);
    const ciphertext = combined.slice(28);

    // Derive decryption key from password
    const decryptionKey = await deriveKeyFromPassword(password, salt);

    // Decrypt the private key
    const decrypted = await crypto.subtle.decrypt(
        { name: "AES-GCM", iv: iv },
        decryptionKey,
        ciphertext
    );

    // Import the private key back
    const privateKeyJwk = JSON.parse(new TextDecoder().decode(decrypted));
    return await crypto.subtle.importKey(
        "jwk",
        privateKeyJwk,
        { name: "ECDH", namedCurve: "P-256" },
        true,
        ["deriveKey", "deriveBits"]
    );
}

async function handleLogin(e) {
    e.preventDefault();
    const username = document.getElementById('login-username').value;
    const passwordRaw = document.getElementById('login-password').value;

    try {
        const passwordHash = await hashPassword(passwordRaw);
        const res = await fetch('/login', {
            method: 'POST',
            body: JSON.stringify({ username, password: passwordHash }),
            headers: { 'Content-Type': 'application/json' }
        });

        if (res.ok) {
            const me = await res.json();
            currentUser = me.username;
            currentUserID = me.id;

            if (me.encrypted_private_key) {
                try {
                    // Decrypt private key
                    sessionPrivateKey = await decryptPrivateKey(me.encrypted_private_key, passwordRaw);
                    console.log("Private key decrypted and cached.");
                } catch (e) {
                    console.error("Failed to decrypt private key:", e);
                    alert("Failed to decrypt private key. Wrong password?");
                    return;
                }

                localStorage.setItem('public_key', me.public_key);
            }

            document.getElementById('login-username').value = '';
            document.getElementById('login-password').value = '';

            document.getElementById('auth-section').style.display = 'none';
            document.getElementById('chat-section').style.display = 'flex';
            document.getElementById('current-username').textContent = currentUser;

            loadChats();
            connectWS();
        } else {
            alert('Login failed');
        }
    } catch (err) {
        console.error(err);
        alert('Error logging in');
    }
}

async function handleSignup(e) {
    e.preventDefault();
    const username = document.getElementById('signup-username').value;
    const passwordRaw = document.getElementById('signup-password').value;

    try {
        const passwordHash = await hashPassword(passwordRaw);

        // Generate ECDH key pair
        const keyPair = await generateKeyPair();

        // Export public key
        const publicKeyRaw = await crypto.subtle.exportKey("jwk", keyPair.publicKey);
        const publicKey = toBase64(new TextEncoder().encode(JSON.stringify(publicKeyRaw)));

        // Encrypt private key with password
        const encryptedPrivateKey = await encryptPrivateKey(keyPair.privateKey, passwordRaw);

        const res = await fetch('/signup', {
            method: 'POST',
            body: JSON.stringify({
                username,
                password: passwordHash,
                public_key: publicKey,
                encrypted_private_key: encryptedPrivateKey
            }),
            headers: { 'Content-Type': 'application/json' }
        });

        if (res.ok) {
            alert('Signup successful! Please login.');
            document.getElementById('signup-username').value = '';
            document.getElementById('signup-password').value = '';
            showTab('login');
        } else {
            alert('Signup failed');
        }
    } catch (err) {
        console.error(err);
        alert('Error signing up');
    }
}

function logout() {
    document.cookie = "user_id=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;";
    document.cookie = "username=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;";
    location.reload();
}

// Chat Management
async function loadChats() {
    try {
        const res = await fetch('/chats');
        const chats = await res.json();
        const list = document.getElementById('chat-list');
        list.innerHTML = '';
        chatKeys = {}; // Reset keys

        if (chats) {
            for (const chat of chats) {
                // Decrypt chat key if we have it
                if (chat.encrypted_key && sessionPrivateKey) {
                    try {
                        const decryptedKey = await decryptAsymmetric(chat.encrypted_key, sessionPrivateKey);
                        chatKeys[chat.id] = decryptedKey;
                        console.log(`Chat ${chat.id} (${chat.name}): Decrypted symmetric key:`, toBase64(decryptedKey));
                    } catch (e) {
                        console.error(`Failed to decrypt key for chat ${chat.id}:`, e);
                    }
                }

                const div = document.createElement('div');
                div.className = 'chat-item';
                div.textContent = chat.name;
                div.onclick = () => selectChat(chat);
                list.appendChild(div);
            }
        }
    } catch (err) {
        console.error(err);
    }
}

async function selectChat(chat) {
    currentChat = chat;
    document.getElementById('no-chat-selected').style.display = 'none';
    document.getElementById('active-chat').style.display = 'flex';
    document.getElementById('participants-sidebar').style.display = 'flex';
    document.getElementById('active-chat-name').textContent = chat.name;

    // Show appropriate button based on ownership
    const deleteBtn = document.getElementById('delete-chat-btn');
    if (chat.owner_id === currentUserID) {
        deleteBtn.textContent = 'Delete Chat';
        deleteBtn.onclick = () => deleteChat(chat.id);
        deleteBtn.style.display = 'block';
    } else {
        deleteBtn.textContent = 'Leave Chat';
        deleteBtn.onclick = leaveChat;
        deleteBtn.style.display = 'block';
    }

    document.getElementById('messages').innerHTML = '';

    // Update active state in sidebar
    document.querySelectorAll('.chat-item').forEach(el => {
        el.classList.remove('active');
        if (el.textContent === chat.name) el.classList.add('active');
    });

    // Load messages
    try {
        const res = await fetch(`/chats/${chat.id}/messages`);
        const messages = await res.json();

        // Prevent race condition: ensure we are still on the same chat
        if (currentChat && currentChat.id === chat.id) {
            if (messages) {
                messages.forEach(appendMessage);
            }
        }
    } catch (err) {
        console.error("Error loading messages:", err);
    }

    // Load participants
    loadParticipants(chat.id, chat.owner_id);
}

async function loadParticipants(chatID, ownerID) {
    try {
        const res = await fetch(`/chats/${chatID}/participants`);
        const participants = await res.json();

        const list = document.getElementById('participants-list');
        list.innerHTML = '';

        if (participants) {
            participants.forEach(participant => {
                const div = document.createElement('div');
                div.className = 'participant-item';
                if (participant.id === ownerID) {
                    div.classList.add('owner');
                }

                const nameSpan = document.createElement('span');
                nameSpan.textContent = participant.username;
                div.appendChild(nameSpan);

                if (participant.id === ownerID) {
                    const badge = document.createElement('span');
                    badge.className = 'owner-badge';
                    badge.textContent = 'OWNER';
                    div.appendChild(badge);
                }

                list.appendChild(div);
            });
        }
    } catch (err) {
        console.error("Error loading participants:", err);
    }
}

function showCreateChat() {
    document.getElementById('create-chat-modal').style.display = 'block';
}

function showInvite() {
    document.getElementById('invite-modal').style.display = 'block';
}

async function deleteChat() {
    if (!currentChat) return;

    if (!confirm(`Are you sure you want to delete chat "${currentChat.name}"? This action cannot be undone.`)) {
        return;
    }

    try {
        const res = await fetch(`/chats/${currentChat.id}`, {
            method: 'DELETE'
        });

        if (res.ok) {
            // UI update will be handled by WebSocket 'chat_deleted' event
            // But we can optimistically clear the view
            document.getElementById('active-chat').style.display = 'none';
            document.getElementById('no-chat-selected').style.display = 'flex';
            document.getElementById('participants-sidebar').style.display = 'none';
            currentChat = null;
        } else {
            const err = await res.text();
            alert('Failed to delete chat: ' + err);
        }
    } catch (err) {
        console.error(err);
        alert('Error deleting chat');
    }
}

function closeModal(id) {
    document.getElementById(id).style.display = 'none';
}

async function generateSymKey() {
    return crypto.getRandomValues(new Uint8Array(32));
}

async function encryptAsymmetric(data, publicKeyBase64) {
    // Import the recipient's public key
    const publicKeyBytes = fromBase64(publicKeyBase64);
    const publicKeyJwk = JSON.parse(new TextDecoder().decode(publicKeyBytes));
    const recipientPublicKey = await crypto.subtle.importKey(
        "jwk",
        publicKeyJwk,
        { name: "ECDH", namedCurve: "P-256" },
        true,
        []
    );

    // Generate an ephemeral key pair for this encryption
    const ephemeralKeyPair = await crypto.subtle.generateKey(
        { name: "ECDH", namedCurve: "P-256" },
        true,
        ["deriveKey"]
    );

    // Derive a shared secret using ECDH
    const sharedSecret = await crypto.subtle.deriveKey(
        {
            name: "ECDH",
            public: recipientPublicKey
        },
        ephemeralKeyPair.privateKey,
        { name: "AES-GCM", length: 256 },
        false,
        ["encrypt"]
    );

    // Encrypt the data with the shared secret
    const iv = crypto.getRandomValues(new Uint8Array(12));
    const encrypted = await crypto.subtle.encrypt(
        { name: "AES-GCM", iv: iv },
        sharedSecret,
        data
    );

    // Export the ephemeral public key
    const ephemeralPublicKeyJwk = await crypto.subtle.exportKey("jwk", ephemeralKeyPair.publicKey);
    const ephemeralPublicKeyBytes = new TextEncoder().encode(JSON.stringify(ephemeralPublicKeyJwk));

    // Combine: ephemeralPublicKeyLength (2 bytes) + ephemeralPublicKey + IV (12 bytes) + ciphertext
    const ephemeralKeyLength = ephemeralPublicKeyBytes.length;
    const combined = new Uint8Array(2 + ephemeralKeyLength + iv.length + encrypted.byteLength);

    // Store length as 2-byte big-endian
    combined[0] = (ephemeralKeyLength >> 8) & 0xFF;
    combined[1] = ephemeralKeyLength & 0xFF;
    combined.set(ephemeralPublicKeyBytes, 2);
    combined.set(iv, 2 + ephemeralKeyLength);
    combined.set(new Uint8Array(encrypted), 2 + ephemeralKeyLength + iv.length);

    return toBase64(combined);
}

async function decryptAsymmetric(encryptedData, privateKey) {
    // Parse the encrypted data
    const combined = fromBase64(encryptedData);

    // Extract ephemeral public key length (2 bytes, big-endian)
    const ephemeralKeyLength = (combined[0] << 8) | combined[1];

    // Extract components
    const ephemeralPublicKeyBytes = combined.slice(2, 2 + ephemeralKeyLength);
    const iv = combined.slice(2 + ephemeralKeyLength, 2 + ephemeralKeyLength + 12);
    const ciphertext = combined.slice(2 + ephemeralKeyLength + 12);

    // Import the ephemeral public key
    const ephemeralPublicKeyJwk = JSON.parse(new TextDecoder().decode(ephemeralPublicKeyBytes));
    const ephemeralPublicKey = await crypto.subtle.importKey(
        "jwk",
        ephemeralPublicKeyJwk,
        { name: "ECDH", namedCurve: "P-256" },
        true,
        []
    );

    // Derive the same shared secret using our private key and the ephemeral public key
    const sharedSecret = await crypto.subtle.deriveKey(
        {
            name: "ECDH",
            public: ephemeralPublicKey
        },
        privateKey,
        { name: "AES-GCM", length: 256 },
        false,
        ["decrypt"]
    );

    // Decrypt the data
    const decrypted = await crypto.subtle.decrypt(
        { name: "AES-GCM", iv: iv },
        sharedSecret,
        ciphertext
    );

    return new Uint8Array(decrypted);
}

async function handleCreateChat(e) {
    e.preventDefault();
    const name = document.getElementById('new-chat-name').value;

    try {
        // Generate symmetric key for the chat
        const symKey = await generateSymKey();
        console.log('Generated symmetric key:', symKey);

        const myPublicKey = localStorage.getItem('public_key');
        if (!myPublicKey) {
            alert('Public key not found. Please login again.');
            return;
        }
        console.log('My public key:', myPublicKey);

        const encryptedKey = await encryptAsymmetric(symKey, myPublicKey);
        console.log('Encrypted key:', encryptedKey);

        const payload = { name, encrypted_key: encryptedKey };
        console.log('Sending payload:', payload);

        const res = await fetch('/chats', {
            method: 'POST',
            body: JSON.stringify(payload),
            headers: { 'Content-Type': 'application/json' }
        });

        console.log('Response status:', res.status);
        if (res.ok) {
            document.getElementById('new-chat-name').value = '';
            closeModal('create-chat-modal');
            loadChats();
        } else {
            const errorText = await res.text();
            console.error('Server error:', errorText);
            alert('Failed to create chat: ' + errorText);
        }
    } catch (err) {
        console.error('Error in handleCreateChat:', err);
        alert('Error creating chat: ' + err.message);
    }
}

async function handleInviteUser(e) {
    e.preventDefault();
    const username = document.getElementById('invite-username').value;
    const chatID = currentChat.id;

    try {
        // 1. Get invitee's public key
        const searchRes = await fetch(`/users/search?q=${username}`);
        const users = await searchRes.json();
        const invitee = users.find(u => u.username === username);
        if (!invitee || !invitee.public_key) {
            alert('User not found or has no public key');
            return;
        }

        // 2. Get the chat's decrypted symmetric key
        const decryptedChatKey = chatKeys[chatID];
        if (!decryptedChatKey) {
            alert('Chat key not found (or not decrypted). Cannot invite.');
            return;
        }

        console.log('Inviting user to chat:', chatID);
        console.log('Decrypted symmetric key to share:', toBase64(decryptedChatKey));

        // 3. Encrypt the symmetric key for the invitee using their public key
        const encryptedForInvitee = await encryptAsymmetric(decryptedChatKey, invitee.public_key);
        console.log('Encrypted for invitee:', encryptedForInvitee);

        const res = await fetch(`/chats/${chatID}/invite`, {
            method: 'POST',
            body: JSON.stringify({ username, encrypted_key: encryptedForInvitee }),
            headers: { 'Content-Type': 'application/json' }
        });

        if (res.ok) {
            document.getElementById('invite-username').value = '';
            alert('User invited');
            closeModal('invite-modal');
        } else {
            alert('Failed to invite user');
        }
    } catch (err) {
        console.error(err);
        alert('Error inviting user');
    }
}

async function leaveChat() {
    if (!currentChat) return;
    if (!confirm('Are you sure you want to leave this chat?')) return;

    try {
        const res = await fetch(`/chats/${currentChat.id}/leave`, {
            method: 'DELETE'
        });

        if (res.ok) {
            currentChat = null;
            currentChatID = null;
            document.getElementById('messages').innerHTML = '';
            document.getElementById('active-chat-name').textContent = '';
            document.getElementById('message-input').disabled = true;
            document.getElementById('send-btn').disabled = true;
            document.getElementById('delete-chat-btn').style.display = 'none';

            // Switch back to no chat selected view
            document.getElementById('active-chat').style.display = 'none';
            document.getElementById('no-chat-selected').style.display = 'flex';
            document.getElementById('participants-sidebar').style.display = 'none';
            loadChats();
        } else {
            const err = await res.text();
            alert('Failed to leave chat: ' + err);
        }
    } catch (err) {
        console.error(err);
        alert('Error leaving chat');
    }
}

let searchTimeout;
async function handleSearchUsers(query) {
    const dropdown = document.getElementById('user-suggestions');
    if (!query) {
        dropdown.style.display = 'none';
        return;
    }

    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(async () => {
        try {
            const res = await fetch(`/users/search?q=${encodeURIComponent(query)}`);
            const users = await res.json();

            // Get current participants to filter them out
            const participantsRes = await fetch(`/chats/${currentChat.id}/participants`);
            const participants = await participantsRes.json();
            const participantUsernames = new Set(participants.map(p => p.username));

            // Filter out current user and existing participants
            const filteredUsers = users.filter(u =>
                u.username !== currentUser && !participantUsernames.has(u.username)
            );

            dropdown.innerHTML = '';

            if (filteredUsers.length > 0) {
                filteredUsers.forEach(user => {
                    const div = document.createElement('div');
                    div.className = 'suggestion-item';
                    div.textContent = user.username;
                    div.onclick = () => selectUser(user.username);
                    dropdown.appendChild(div);
                });
                dropdown.style.display = 'block';
            } else {
                dropdown.style.display = 'none';
            }
        } catch (err) {
            console.error(err);
        }
    }, 300);
}

function selectUser(username) {
    document.getElementById('invite-username').value = username;
    document.getElementById('user-suggestions').style.display = 'none';
}

// Message Encryption/Decryption
async function encryptMessage(content, symmetricKey) {
    // Convert symmetric key (Uint8Array) to CryptoKey
    const key = await crypto.subtle.importKey(
        "raw",
        symmetricKey,
        { name: "AES-GCM", length: 256 },
        false,
        ["encrypt"]
    );

    // Generate IV
    const iv = crypto.getRandomValues(new Uint8Array(12));

    // Encrypt the message
    const contentBytes = new TextEncoder().encode(content);
    const encrypted = await crypto.subtle.encrypt(
        { name: "AES-GCM", iv: iv },
        key,
        contentBytes
    );

    // Combine IV + ciphertext
    const combined = new Uint8Array(iv.length + encrypted.byteLength);
    combined.set(iv, 0);
    combined.set(new Uint8Array(encrypted), iv.length);

    return toBase64(combined);
}

async function decryptMessage(encryptedContent, symmetricKey) {
    // Convert symmetric key (Uint8Array) to CryptoKey
    const key = await crypto.subtle.importKey(
        "raw",
        symmetricKey,
        { name: "AES-GCM", length: 256 },
        false,
        ["decrypt"]
    );

    // Parse encrypted data
    const combined = fromBase64(encryptedContent);
    const iv = combined.slice(0, 12);
    const ciphertext = combined.slice(12);

    // Decrypt the message
    const decrypted = await crypto.subtle.decrypt(
        { name: "AES-GCM", iv: iv },
        key,
        ciphertext
    );

    return new TextDecoder().decode(decrypted);
}

// WebSocket & Messaging
function connectWS() {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${protocol}//${location.host}/ws`);

    ws.onmessage = (event) => {
        const msg = JSON.parse(event.data);
        if (msg.type === 'new_chat') {
            loadChats();
            // If we're viewing a chat, refresh participants list
            if (currentChat) {
                loadParticipants(currentChat.id, currentChat.owner_id);
            }
            return;
        }
        if (msg.type === 'chat_deleted') {
            // Always reload the chat list to remove the deleted chat
            loadChats();

            // If the deleted chat is the one currently being viewed, clear the view
            if (currentChat && currentChat.id === msg.chat_id) {
                document.getElementById('active-chat').style.display = 'none';
                document.getElementById('no-chat-selected').style.display = 'flex';
                document.getElementById('participants-sidebar').style.display = 'none';
                currentChat = null;
                alert('This chat has been deleted by the owner.');
            }
            return;
        }
        if (msg.type === 'participant_left') {
            // Ignore if I'm the one who left (handled by leaveChat function)
            if (msg.user_id === currentUserID) {
                return;
            }
            // Refresh participant list if viewing this chat
            if (currentChat && currentChat.id === msg.chat_id) {
                loadParticipants(msg.chat_id, currentChat.owner_id);
            }
            return;
        }
        if (currentChat && msg.chat_id === currentChat.id) {
            appendMessage(msg);
        }
    };

    ws.onclose = () => {
        console.log("WS disconnected. Reconnecting...");
        setTimeout(connectWS, 3000);
    };
}

async function sendMessage(e) {
    e.preventDefault();
    const input = document.getElementById('message-input');
    const content = input.value;

    if (ws && currentChat) {
        try {
            // Get the symmetric key for this chat
            const symmetricKey = chatKeys[currentChat.id];
            if (!symmetricKey) {
                alert('Chat key not available. Cannot send encrypted message.');
                return;
            }

            // Encrypt the message
            const encryptedContent = await encryptMessage(content, symmetricKey);

            const msg = {
                chat_id: currentChat.id,
                user_id: 0, // Backend handles this
                content: encryptedContent
            };
            ws.send(JSON.stringify(msg));
            input.value = '';
        } catch (err) {
            console.error('Error encrypting message:', err);
            alert('Failed to encrypt message');
        }
    }
}

async function appendMessage(msg) {
    const div = document.createElement('div');
    const isMe = msg.username === currentUser;
    div.className = `message ${isMe ? 'sent' : 'received'}`;

    const meta = document.createElement('div');
    meta.className = 'message-meta';

    const date = new Date(msg.created_at);
    const timeStr = date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    meta.textContent = `${msg.username} • ${timeStr}`;

    const content = document.createElement('div');

    // Decrypt the message content
    try {
        const symmetricKey = chatKeys[msg.chat_id];
        if (symmetricKey) {
            const decryptedContent = await decryptMessage(msg.content, symmetricKey);
            content.textContent = decryptedContent;
        } else {
            content.textContent = '[Encrypted - key not available]';
            console.warn('No symmetric key available for chat:', msg.chat_id);
        }
    } catch (err) {
        console.error('Error decrypting message:', err);
        content.textContent = '[Decryption failed]';
    }

    div.appendChild(meta);
    div.appendChild(content);

    const container = document.getElementById('messages');
    container.appendChild(div);
    container.scrollTop = container.scrollHeight;
}

// Close modals when clicking outside
window.onclick = function (event) {
    if (event.target.classList.contains('modal')) {
        event.target.style.display = "none";
    }
}
