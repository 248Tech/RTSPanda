/* global chrome */

function isCameraPage(url) {
  try {
    const parsed = new URL(url);
    return parsed.pathname.startsWith("/cameras/");
  } catch (_) {
    return false;
  }
}

async function togglePiPInTab(tabId) {
  const injection = await chrome.scripting.executeScript({
    target: { tabId },
    func: async () => {
      const result = { ok: false, message: "" };
      try {
        const video =
          document.querySelector("video[aria-label='Camera stream']") ||
          document.querySelector("video");
        if (!video) {
          result.message = "No camera video element found on this page.";
          return result;
        }

        if (!document.pictureInPictureEnabled || video.disablePictureInPicture) {
          result.message = "Picture-in-Picture is not available for this video.";
          return result;
        }

        if (document.pictureInPictureElement === video) {
          await document.exitPictureInPicture();
          return { ok: true, message: "PiP closed." };
        }

        if (video.paused) {
          try {
            await video.play();
          } catch (_) {
            // Ignore play errors and continue attempting PiP request.
          }
        }

        await video.requestPictureInPicture();
        return { ok: true, message: "PiP opened." };
      } catch (error) {
        result.message = error && error.message ? error.message : "Failed to toggle PiP.";
        return result;
      }
    }
  });

  const output = injection && injection[0] ? injection[0].result : null;
  return output && output.ok;
}

chrome.action.onClicked.addListener(async (tab) => {
  if (!tab || typeof tab.id !== "number") {
    return;
  }

  if (!tab.url || !isCameraPage(tab.url)) {
    await chrome.action.setBadgeBackgroundColor({ color: "#ef4444", tabId: tab.id });
    await chrome.action.setBadgeText({ text: "ERR", tabId: tab.id });
    setTimeout(() => {
      chrome.action.setBadgeText({ text: "", tabId: tab.id });
    }, 1500);
    return;
  }

  try {
    const ok = await togglePiPInTab(tab.id);
    if (!ok) {
      await chrome.action.setBadgeBackgroundColor({ color: "#ef4444", tabId: tab.id });
      await chrome.action.setBadgeText({ text: "ERR", tabId: tab.id });
      setTimeout(() => {
        chrome.action.setBadgeText({ text: "", tabId: tab.id });
      }, 1500);
      return;
    }
    await chrome.action.setBadgeBackgroundColor({ color: "#16a34a", tabId: tab.id });
    await chrome.action.setBadgeText({ text: "PIP", tabId: tab.id });
    setTimeout(() => {
      chrome.action.setBadgeText({ text: "", tabId: tab.id });
    }, 1200);
  } catch (_) {
    await chrome.action.setBadgeBackgroundColor({ color: "#ef4444", tabId: tab.id });
    await chrome.action.setBadgeText({ text: "ERR", tabId: tab.id });
    setTimeout(() => {
      chrome.action.setBadgeText({ text: "", tabId: tab.id });
    }, 1500);
  }
});
