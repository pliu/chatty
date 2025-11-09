let currentUser = null;
let currentChat = null;
let ws = null;
// Check for existing session
window.onload = function () {
    const cookies = document.cookie.split(';');
    let username = null;
    for (let cookie of cookies) {
        const [name, value] = cookie.trim().split('=');
        if (name === 'username') {
            username = value;
            break;
        }
    }

    if (username) {
        currentUser = username;
        document.getElementById('auth-section').style.display = 'none';
        document.getElementById('chat-section').style.display = 'flex';
        document.getElementById('current-username').textContent = username;
        loadChats();
        connectWS();
    }
};
// Auth
function showTab(tab) {
    document.getElementById('login-form').style.display = tab === 'login' ? 'flex' : 'none';
    document.getElementById('signup-form').style.display = tab === 'signup' ? 'flex' : 'none';
}

async function handleLogin(e) {
    e.preventDefault();
    const username = document.getElementById('login-username').value;
    const password = document.getElementById('login-password').value;

    try {
        const res = await fetch('/login', {
            method: 'POST',
            body: JSON.stringify({ username, password }),
            headers: { 'Content-Type': 'application/json' }
        });

        if (res.ok) {
            currentUser = username;
            document.getElementById('auth-section').style.display = 'none';
            document.getElementById('chat-section').style.display = 'flex';
            document.getElementById('current-username').textContent = username;
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
    const password = document.getElementById('signup-password').value;

    try {
        const res = await fetch('/signup', {
            method: 'POST',
            body: JSON.stringify({ username, password }),
            headers: { 'Content-Type': 'application/json' }
        });

        if (res.ok) {
            alert('Signup successful! Please login.');
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

        if (chats) {
            chats.forEach(chat => {
                const div = document.createElement('div');
                div.className = 'chat-item';
                div.textContent = chat.name;
                div.onclick = () => selectChat(chat);
                list.appendChild(div);
            });
        }
    } catch (err) {
        console.error(err);
    }
}

async function selectChat(chat) {
    currentChat = chat;
    document.getElementById('no-chat-selected').style.display = 'none';
    document.getElementById('active-chat').style.display = 'flex';
    document.getElementById('active-chat-name').textContent = chat.name;
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
}

function showCreateChat() {
    document.getElementById('create-chat-modal').style.display = 'block';
}

function showInvite() {
    document.getElementById('invite-modal').style.display = 'block';
}

function closeModal(id) {
    document.getElementById(id).style.display = 'none';
}

async function handleCreateChat(e) {
    e.preventDefault();
    const name = document.getElementById('new-chat-name').value;

    try {
        const res = await fetch('/chats', {
            method: 'POST',
            body: JSON.stringify({ name }),
            headers: { 'Content-Type': 'application/json' }
        });

        if (res.ok) {
            closeModal('create-chat-modal');
            loadChats();
            document.getElementById('new-chat-name').value = '';
        } else {
            alert('Failed to create chat');
        }
    } catch (err) {
        console.error(err);
    }
}

async function handleInvite(e) {
    e.preventDefault();
    const username = document.getElementById('invite-username').value;

    try {
        const res = await fetch(`/chats/${currentChat.id}/invite`, {
            method: 'POST',
            body: JSON.stringify({ username }),
            headers: { 'Content-Type': 'application/json' }
        });

        if (res.ok) {
            alert('User invited!');
            closeModal('invite-modal');
            document.getElementById('invite-username').value = '';
        } else {
            alert('Failed to invite user');
        }
    } catch (err) {
        console.error(err);
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

            dropdown.innerHTML = '';
            if (users && users.length > 0) {
                users.forEach(user => {
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

// WebSocket & Messaging
function connectWS() {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${protocol}//${location.host}/ws`);

    ws.onmessage = (event) => {
        const msg = JSON.parse(event.data);
        if (msg.type === 'new_chat') {
            loadChats();
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

function sendMessage(e) {
    e.preventDefault();
    const input = document.getElementById('message-input');
    const content = input.value;

    if (ws && currentChat) {
        const msg = {
            chat_id: currentChat.id,
            user_id: 0, // Backend handles this
            content: content
        };
        ws.send(JSON.stringify(msg));
        input.value = '';
    }
}

function appendMessage(msg) {
    const div = document.createElement('div');
    const isMe = msg.username === currentUser;
    div.className = `message ${isMe ? 'sent' : 'received'}`;

    const meta = document.createElement('div');
    meta.className = 'message-meta';

    const date = new Date(msg.created_at);
    const timeStr = date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    meta.textContent = `${msg.username} • ${timeStr}`;

    const content = document.createElement('div');
    content.textContent = msg.content;

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
