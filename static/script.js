const recordBtn = document.getElementById("recordBtn");
const stopBtn = document.getElementById("stopBtn");
const statusElement = document.getElementById("recordingStatus");
const diskStatsElement = document.getElementById("diskStats");

const daysInput = document.getElementById("daysInput");
const deleteBtn = document.getElementById("deleteBtn");
const cleanupStatusElement = document.getElementById("cleanupStatus");

const serverUptime = document.getElementById("serverUptime");
const recordingUptime = document.getElementById("recordingUptime");

// const resetBtn = document.getElementById('resetBtn');

let serverStartTimeMillis = -1;
let currentRecordingStartTimeMillis = -1;

async function sendDeleteCommand() {
    const days = daysInput.value;

    if (days === "" || isNaN(days) || parseInt(days) < 0) {
        cleanupStatusElement.textContent =
            "Status: Please enter a valid number of days (0 or more).";
        return;
    }

    cleanupStatusElement.textContent = `Status: Deleting files older than ${days} days...`;
    deleteBtn.disabled = true;

    let response = null;

    try {
        response = await fetch("/delete", {
            method: "POST",
            headers: {
                "Content-Type": "application/x-www-form-urlencoded",
            },
            body: "days=" + days,
        });

        const result = await response.text();

        if (response.ok) {
            cleanupStatusElement.textContent = `Status: ${result}`;
        } else {
            cleanupStatusElement.textContent = `Status: Error (${response.status}) - ${result}`;
        }
    } catch (error) {
        console.error("Fetch error (delete):", error);
        cleanupStatusElement.textContent = `Status: Network Error - ${error.message}`;
    } finally {
        deleteBtn.disabled = false;
    }
}

async function sendAction(action) {
    statusElement.textContent = `Status: Sending '${action}' command...`;

    try {
        const response = await fetch("/record", {
            method: "POST",
            headers: {
                "Content-Type": "application/x-www-form-urlencoded",
            },
            body: "action=" + action, // action=start, stop
        });

        const result = await response.text();

        if (response.ok) {
            statusElement.textContent = `Status: ${result}`;
        } else {
            statusElement.textContent = `Status: Error (${response.status}) - ${result}`;
        }
    } catch (error) {
        console.error("Fetch error:", error);
        statusElement.textContent = `Status: Network Error - ${error.message}`;
    }
}

async function fetchDiskStatistics() {
    try {
        const response = await fetch("/statistics");

        if (!response.ok) {
            throw new Error(`HTTP error! Status: ${response.status}`);
        }

        const data = await response.json();

        const statsHTML = `
                <div class="stat-item">
                    <span class="stat-label">Total Space:</span>
                    <span class="stat-value">${data.totalSpaceFormatted}</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">Free Space:</span>
                    <span class="stat-value">${data.freeSpaceFormatted}</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">Usable Space:</span>
                    <span class="stat-value">${data.usableSpaceFormatted}</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">Space Used:</span>
                    <span class="stat-value">${calculateUsedSpacePercentage(
                        data
                    )}</span>
                </div>
            `;

        diskStatsElement.innerHTML = statsHTML;
        serverStartTimeMillis = Number(data.serverStartTimeMillis);
        currentRecordingStartTimeMillis = Number(data.recordingStartTimeMillis);
    } catch (error) {
        console.error("Error fetching disk statistics:", error);
        diskStatsElement.innerHTML = `<p>Error loading disk statistics: ${error.message}</p>`;
    }
}

function updateUptimesDisplay() {
    const currentTimeMillis = Date.now();

    if (serverStartTimeMillis !== -1) {
        const serverUptimeMillis = currentTimeMillis - serverStartTimeMillis;
        serverUptime.textContent = formatDurationJS(serverUptimeMillis);
    } else {
        serverUptime.textContent = "Loading...";
    }

    if (currentRecordingStartTimeMillis !== -1) {
        const recordingUptimeMillis =
            currentTimeMillis - currentRecordingStartTimeMillis;
        recordingUptime.textContent = formatDurationJS(recordingUptimeMillis);
        statusElement.textContent = `Status: Recording`;
    } else {
        recordingUptime.textContent = "Idle";
    }
}

function calculateUsedSpacePercentage(data) {
    const total = data.totalSpace;
    const free = data.freeSpace;
    const used = total - free;
    const percentage = Math.round((used / total) * 100);
    return `${percentage}% (${formatBytes(used)})`;
}

function formatDurationJS(millis) {
    if (millis < 0) return "N/A";

    const totalSeconds = Math.floor(millis / 1000);
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const seconds = totalSeconds % 60;

    const parts = [];
    if (hours > 0) parts.push(hours + " h");
    if (minutes > 0 || hours > 0) parts.push(minutes + " m");
    if (seconds > 0 || totalSeconds === 0) parts.push(seconds + " s");

    return parts.join(" ");
}

function formatBytes(bytes) {
    if (bytes === 0) return "0 B";

    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

recordBtn.addEventListener("click", () => sendAction("start"));
stopBtn.addEventListener("click", () => sendAction("stop"));
deleteBtn.addEventListener("click", () => sendDeleteCommand());
// resetBtn.addEventListener('click', () => sendReset('true'));

fetchDiskStatistics();

// Refresh every 60 seconds
setInterval(fetchDiskStatistics, 60000);
setInterval(updateUptimesDisplay, 1000);
