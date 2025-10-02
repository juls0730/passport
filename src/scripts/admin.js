// idfk what this variable capitalization is, it's a mess
let modalContainer = document.getElementById("modal-container");
let modal = modalContainer.querySelector("div");
let pageElement = document.getElementById("blur-target");
let iconUploadInput = document.getElementById("icon-upload");
let targetCategoryID = null;
let activeModal = null;

let teleportStorage = document.getElementById("teleport-storage");
let confirmActions = document.getElementById("confirm-actions");
let selectIconButton = document.getElementById("select-icon-button");

document.addEventListener("DOMContentLoaded", () => {
    modalContainer.classList.remove("hidden");
    modalContainer.classList.add("flex");
});

/**
 * Submits a form to the given URL
 * @param {Event} event - The event that triggered the function
 * @param {string} url - The URL to submit the form to
 * @param {"category" | "link"} target - The target to close the modal for
 * @returns {Promise<void>}
 */
async function submitRequest(event, url, target) {
    event.preventDefault();
    let data = new FormData(event.target);

    let res = await fetch(url, {
        method: "POST",
        body: data,
    });

    if (res.status === 201) {
        closeModal(target);
        document.getElementById(`${target}-form`).reset();
        location.reload();
    } else {
        let json = await res.json();
        document.getElementById(`${target}-message`).innerText = json.message;
    }
}

/**
 * Adds an event listener for the given from to error check after the first submit
 * @param {"category" | "link"} form - The form to initialize
 * @returns {void}
 */
function addErrorListener(form) {
    document
        .getElementById(`${form}-form`)
        .querySelector("button")
        .addEventListener("click", (event) => {
            event.target.parentElement
                .querySelectorAll("[required]")
                .forEach((el) => {
                    el.classList.add("invalid:border-[#861024]!");
                });
        });
}

/**
 * Currently editing link or category
 * @typedef {Object} actionButtonObj
 * @property {string} clickAction - The function to be called when this button is clicked
 * @property {string} label - The label of the button
 */

/**
 * Clones the edit actions template and returns it
 * @param {[actionButtonObj, actionButtonObj]} primaryActions - The primary actions to clone
 * @returns {HTMLElement} The cloned edit actions element
 */
function cloneEditActions(primaryActions) {
    let editActions = document
        .getElementById("template-edit-actions")
        .cloneNode(true);
    editActions.removeAttribute("id");
    editActions.classList.remove("hidden");

    let i = 0;
    for (i = 0; i < primaryActions.length; i++) {
        let actionButtonObj = primaryActions[i];

        let actionButton = editActions.querySelector(
            `div[data-primary-actions] button:nth-child(${i + 1})`
        );
        actionButton.setAttribute("onclick", actionButtonObj.clickAction);
        actionButton.setAttribute("aria-label", actionButtonObj.label);
    }

    return editActions;
}

addErrorListener("link");
document
    .getElementById("link-form")
    .addEventListener("submit", async (event) => {
        event.preventDefault();
        let data = new FormData(event.target);

        let res = await fetch(`/api/category/${targetCategoryID}/link`, {
            method: "POST",
            body: data,
        });

        if (res.status === 201) {
            let json = await res.json();

            let category = document.getElementById(
                `${targetCategoryID}_category`
            );
            let linkGrid = category.nextElementSibling;

            let newLinkCard = document
                .getElementById("template-link-card")
                .cloneNode(true);

            newLinkCard.classList.remove("hidden");
            newLinkCard.classList.add("link-card", "admin", "relative");

            let newLinkImgElement = newLinkCard.querySelector(
                "div[data-img-container] img"
            );

            newLinkImgElement.src = await processFile(data.get("icon"));
            newLinkImgElement.alt = data.get("name");

            newLinkCard.querySelector("h3").textContent = data.get("name");
            newLinkCard.querySelector("p").textContent =
                data.get("description");

            newLinkCard.setAttribute("id", `${json.link.id}_link`);

            let editActions = cloneEditActions([
                {
                    clickAction: "editLink(this)",
                    label: "Edit link",
                },
                {
                    clickAction: "deleteLink(this)",
                    label: "Delete link",
                },
            ]);

            editActions.classList.add("absolute", "right-1", "top-1");

            newLinkCard.appendChild(editActions);

            // append the card as the second to last element
            linkGrid.insertBefore(newLinkCard, linkGrid.lastElementChild);
            closeModal("link");

            // after the close animation plays
            setTimeout(() => {
                document.getElementById(`link-form`).reset();
            }, 300);
        } else {
            let json = await res.json();
            document.getElementById(`link-message`).innerText = json.message;
        }
    });

addErrorListener("category");
document
    .getElementById("category-form")
    .addEventListener("submit", async (event) => {
        event.preventDefault();
        let data = new FormData(event.target);

        let res = await fetch(`/api/category`, {
            method: "POST",
            body: data,
        });

        if (res.status === 201) {
            let json = await res.json();

            let newCategory = document
                .getElementById("template-category")
                .cloneNode(true);

            let linkGrid = newCategory.querySelector("div:nth-child(2)");
            let categoryHeader = newCategory.querySelector(".category-header");
            categoryHeader.setAttribute("id", `${json.category.id}_category`);
            categoryHeader.querySelector("h2").textContent = json.category.name;

            let editActions = cloneEditActions([
                {
                    clickAction: "editCategory(this)",
                    label: "Edit category",
                },
                {
                    clickAction: "deleteCategory(this)",
                    label: "Delete category",
                },
            ]);

            editActions.classList.add("pl-2");

            categoryHeader.appendChild(editActions);

            let categoryImg = categoryHeader.querySelector(".category-img");

            categoryImg.querySelector("img").src = await processFile(
                data.get("icon")
            );

            linkGrid
                .querySelector("div")
                .setAttribute(
                    "onclick",
                    `openModal('link', ${json.category.id})`
                );

            let addCategoryButton = document.getElementById(
                "add-category-button"
            );
            addCategoryButton.parentElement.insertBefore(
                categoryHeader,
                addCategoryButton
            );
            addCategoryButton.parentElement.insertBefore(
                linkGrid,
                addCategoryButton
            );

            closeModal("category");

            // after the close animation plays
            setTimeout(() => {
                document.getElementById(`category-form`).reset();
            }, 300);
        } else {
            let json = await res.json();
            document.getElementById(`category-message`).innerText =
                json.message;
        }
    });

// when the background is clicked, close the modal
modalContainer.addEventListener("click", (event) => {
    if (event.target === modalContainer) {
        closeModal();
    }
});

function selectIcon() {
    iconUploadInput.click();
}

/**
 * Processes a file and returns a data URL.
 * @param {File} file The file to process.
 * @returns {Promise<string>} A promise that resolves to a data URL.
 */
async function processFile(file) {
    let reader = new FileReader();
    return new Promise((resolve) => {
        if (file.type === "image/svg+xml") {
            reader.addEventListener("load", async (event) => {
                let svgString = event.target.result;

                svgString = svgString.replaceAll(
                    "currentColor",
                    "oklch(87% 0.015 286)"
                );

                // turn svgString into a data URL
                resolve(
                    "data:image/svg+xml;base64," +
                        btoa(unescape(encodeURIComponent(svgString)))
                );
            });

            reader.readAsText(file);
        } else {
            // these should be jpg, png, or webp
            // make a DataURL out of it
            reader.addEventListener("load", async (event) => {
                resolve(event.target.result);
            });

            reader.readAsDataURL(file);
        }
    });
}

let targetedImageElement = null;
iconUploadInput.addEventListener("change", async (event) => {
    let file = event.target.files[0];
    if (file === null) {
        return;
    }

    if (targetedImageElement === null) {
        throw new Error(
            "icon upload element was clicked, but no target image element was set"
        );
    }

    let dataURL = await processFile(file);
    targetedImageElement.src = dataURL;
});

function openModal(modalKind, categoryID) {
    activeModal = modalKind;
    targetCategoryID = categoryID;

    pageElement.style.filter = "blur(20px)";
    document.getElementById(modalKind + "-contents").classList.remove("hidden");

    modalContainer.classList.add("is-visible");
    modal.classList.add("is-visible");

    if (document.getElementById(modalKind + "-form") !== null) {
        document.getElementById(modalKind + "-form").reset();
    }
}

function closeModal() {
    pageElement.style.filter = "";

    modalContainer.classList.remove("is-visible");
    modal.classList.remove("is-visible");

    setTimeout(() => {
        document
            .getElementById(activeModal + "-contents")
            .classList.add("hidden");
        activeModal = null;
    }, 300);

    if (document.getElementById(activeModal + "-form") !== null) {
        document
            .getElementById(activeModal + "-form")
            .querySelectorAll("[required]")
            .forEach((el) => {
                el.classList.remove("invalid:border-[#861024]!");
            });
    }

    targetCategoryID = null;
}

/**
 * Currently editing link or category
 * @typedef {Object} currentlyEditingObj
 * @property {"link" | "category" | undefined} type - The type of the currently editing element
 * @property {string | undefined} linkID - The ID of the link we are currently editing if we are editing a link
 * @property {string | undefined} categoryID - The ID of the category we are currently editing, or that the link belongs to
 * @property {string | undefined} originalText - The original text of the currently editing element
 * @property {string | undefined} originalDescription - The original description of the currently editing element
 * @property {string | undefined} icon - The original icon of the currently editing element
 * @property {Function | undefined} cleanup - The cleanup function for the currently editing element
 */

/** @type {currentlyEditingObj} */
let currentlyEditing = {};

/**
 * Teleports the upload overlay to the given image node
 * @param {HTMLElement} element The node to teleport into the destination
 * @param {HTMLElement} destination The image node to teleport the upload overlay into
 * @returns {HTMLElement} A reference to the teleported element
 */
function teleportElement(element, destination) {
    destination.appendChild(element);
}

function unteleportElement(element) {
    teleportElement(element, teleportStorage);
}

function confirmEdit() {
    if (currentlyEditing.cleanup !== undefined) {
        // this function could be called via deleting something, which doesn't have a cleanup function
        currentlyEditing.cleanup();
    }

    switch (currentlyEditing.type) {
        case "link":
            confirmLinkEdit();
            break;
        case "category":
            confirmCategoryEdit();
            break;
        default:
            console.error("Unknown currentlyEditing type");
            break;
    }
}

function cancelEdit() {
    if (currentlyEditing.cleanup !== undefined) {
        // this function could be called via deleting something, which doesn't have a cleanup function
        currentlyEditing.cleanup();
    }

    switch (currentlyEditing.type) {
        case "link":
            cancelLinkEdit();
            break;
        case "category":
            cancelCategoryEdit(currentlyEditing.originalText);
            break;
        default:
            console.error("Unknown currentlyEditing type");
            break;
    }

    currentlyEditing = {};
}

/**
 * Edits the link with the given html element
 * @param {HTMLElement} target The target element that was clicked
 */
function editLink(target) {
    let startTime = performance.now();

    // we do it in this dynamic way so that if we add a new link without refreshing the page, it still works
    let linkEl = target.closest(".link-card");
    let linkID = parseInt(linkEl.id);
    let categoryID = parseInt(linkEl.parentElement.previousElementSibling.id);

    if (currentlyEditing.linkID !== undefined) {
        // cancel the edit if it's already in progress
        cancelEdit();
    }

    let linkImg = linkEl.querySelector("div[data-img-container] img");
    let linkName = linkEl.querySelector("div[data-text-container] h3");
    let linkDesc = linkEl.querySelector("div[data-text-container] p");
    let editActions = linkEl.querySelector("[data-edit-actions]");

    currentlyEditing = {
        type: "link",
        linkID: linkID,
        categoryID: categoryID,
        originalText: linkName.textContent,
        originalDescription: linkDesc.textContent,
        icon: linkImg.src,
    };

    if (!currentlyEditing.linkID || !currentlyEditing.categoryID) {
        throw new Error("failed to find link ID or category ID");
    }

    iconUploadInput.accept = "image/*";
    targetedImageElement = linkImg;

    teleportElement(selectIconButton, linkImg.parentElement);
    teleportElement(confirmActions, editActions);

    editActions.querySelector("div[data-primary-actions]").style.display =
        "none";

    requestAnimationFrame(() => {
        currentlyEditing.cleanup = replaceWithResizableTextarea([
            { targetEl: linkName, fill: false },
            { targetEl: linkDesc },
        ]);
        // by adding a delay, we dont block the UI
        setTimeout(() => {
            linkEl.querySelector("textarea").focus();
        }, 0);
    });
}

async function confirmLinkEdit() {
    let linkEl = document.getElementById(`${currentlyEditing.linkID}_link`);
    let linkNameInput = linkEl.querySelector("textarea");
    let linkDescInput = linkNameInput.nextElementSibling;

    linkNameInput.value = linkNameInput.value.trim();
    linkDescInput.value = linkDescInput.value.trim();
    if (linkNameInput.value === "") {
        return;
    }

    let formData = new FormData();
    if (linkNameInput.value !== currentlyEditing.originalText) {
        formData.append("name", linkNameInput.value);
    }

    if (linkDescInput.value !== currentlyEditing.originalDescription) {
        formData.append("description", linkDescInput.value);
    }

    if (iconUploadInput.files.length > 0) {
        formData.append("icon", iconUploadInput.files[0]);
    }

    // nothing to update
    if (
        formData.get("name") === null &&
        formData.get("description") === null &&
        formData.get("icon") === null
    ) {
        return;
    }

    let res = await fetch(
        `/api/category/${currentlyEditing.categoryID}/link/${currentlyEditing.linkID}`,
        {
            method: "PATCH",
            body: formData,
        }
    );

    if (res.status === 200) {
        iconUploadInput.value = "";

        currentlyEditing.icon = undefined;
        cancelLinkEdit(linkNameInput.value, linkDescInput.value);
        currentlyEditing = {};
    } else {
        console.error("Failed to edit category");
    }
}

function cancelLinkEdit(
    text = currentlyEditing.originalText,
    description = currentlyEditing.originalDescription
) {
    let linkEl = document.getElementById(`${currentlyEditing.linkID}_link`);
    let linkInput = linkEl.querySelector("textarea");
    let linkTextarea = linkInput.nextElementSibling;
    let linkImg = linkEl.querySelector("div[data-img-container] img");
    let editActions = linkEl.querySelector("[data-edit-actions]");

    if (currentlyEditing.icon !== undefined) {
        linkImg.src = currentlyEditing.icon;
    }

    editActions.querySelector("div[data-primary-actions]").style.display = "";

    // teleport the teleported elements back to the body for literally safe keeping
    unteleportElement(selectIconButton);
    unteleportElement(confirmActions);

    restoreElementFromInput(linkInput, text);
    restoreElementFromInput(linkTextarea, description);

    currentlyEditing = {};
    targetedImageElement = null;
}

/**
 * Deletes the link with the given html element
 * @param {HTMLElement} target The target element that was clicked
 */
function deleteLink(target) {
    // we do it in this dynamic way so that if we add a new link without refreshing the page, it still works
    let linkEl = target.closest(".link-card");
    let linkID = parseInt(linkEl.id);
    let categoryID = parseInt(linkEl.parentElement.previousElementSibling.id);

    if (currentlyEditing.linkID !== undefined) {
        // cancel the edit if it's already in progress
        cancelEdit();
    }

    currentlyEditing.linkID = linkID;
    currentlyEditing.categoryID = categoryID;

    let linkNameSpan = document.getElementById("link-name");
    linkNameSpan.textContent = linkEl.querySelector("h3").textContent;

    openModal("link-delete");
}

async function confirmDeleteLink() {
    let res = await fetch(
        `/api/category/${currentlyEditing.categoryID}/link/${currentlyEditing.linkID}`,
        {
            method: "DELETE",
        }
    );

    if (res.status === 200) {
        let linkEl = document.getElementById(`${currentlyEditing.linkID}_link`);
        linkEl.remove();

        closeModal();
        currentlyEditing = {};
    }
}

/**
 * Edits the category with the given html element
 * @param {HTMLElement} target The target element that was clicked
 */
function editCategory(target) {
    let categoryEl = target.closest(".category-header");
    let categoryID = parseInt(categoryEl.id);

    if (currentlyEditing.linkID !== undefined) {
        // cancel the edit if it's already in progress
        cancelEdit();
    }

    let categoryName = categoryEl.querySelector("h2");
    let categoryIcon = categoryEl.querySelector("div[data-img-container] img");
    let editActions = categoryEl.querySelector("[data-edit-actions]");

    currentlyEditing = {
        type: "category",
        categoryID: categoryID,
        originalText: categoryName.textContent,
        icon: categoryIcon.src,
    };

    if (!currentlyEditing.categoryID) {
        throw new Error("failed to find category ID");
    }

    iconUploadInput.accept = "image/svg+xml";
    targetedImageElement = categoryIcon;

    teleportElement(selectIconButton, categoryIcon.parentElement);
    teleportElement(confirmActions, editActions);

    editActions.querySelector("div[data-primary-actions]").style.display =
        "none";

    requestAnimationFrame(() => {
        currentlyEditing.cleanup = replaceWithResizableTextarea([
            { targetEl: categoryName, fill: false },
        ]);
        // by adding a delay, we dont block the UI
        setTimeout(() => {
            categoryEl.querySelector("textarea").focus();
        }, 0);
    });
}

async function confirmCategoryEdit() {
    let categoryEl = document.getElementById(
        `${currentlyEditing.categoryID}_category`
    );
    let categoryInput = categoryEl.querySelector("textarea");

    if (categoryInput.value === "") {
        return;
    }

    categoryInput.value = categoryInput.value.trim();

    let formData = new FormData();
    if (categoryInput.value !== currentlyEditing.originalText) {
        formData.append("name", categoryInput.value);
    }

    if (iconUploadInput.files.length > 0) {
        formData.append("icon", iconUploadInput.files[0]);
    }

    // nothing to update
    if (formData.get("name") === null && formData.get("icon") === null) {
        return;
    }

    let res = await fetch(`/api/category/${currentlyEditing.categoryID}`, {
        method: "PATCH",
        body: formData,
    });

    if (res.status === 200) {
        iconUploadInput.value = "";

        currentlyEditing.icon = undefined;

        cancelCategoryEdit(categoryInput.value);

        currentlyEditing = {};
    } else {
        console.error("Failed to edit category");
    }
}

function cancelCategoryEdit(text = currentlyEditing.originalText) {
    let categoryEl = document.getElementById(
        `${currentlyEditing.categoryID}_category`
    );

    let categoryInput = categoryEl.querySelector("textarea");
    let categoryIcon = categoryEl.querySelector(".category-img img");
    let editActions = categoryEl.querySelector("[data-edit-actions]");

    if (currentlyEditing.icon !== undefined) {
        categoryIcon.src = currentlyEditing.icon;
    }

    unteleportElement(selectIconButton);
    unteleportElement(confirmActions);

    editActions.querySelector("div[data-primary-actions]").style.display = "";

    restoreElementFromInput(categoryInput, text);

    currentlyEditing = {};
    targetedImageElement = null;
}

/**
 * Deletes the category with the given html element
 * @param {HTMLElement} target The target element that was clicked
 */
function deleteCategory(target) {
    let categoryEl = target.closest(".category-header");

    if (currentlyEditing.categoryID !== undefined) {
        // cancel the edit if it's already in progress
        cancelEdit();
    }

    let categoryID = parseInt(categoryEl.id);

    currentlyEditing.categoryID = categoryID;

    let categoryNameSpan = document.getElementById("category-name");
    categoryNameSpan.textContent = categoryEl.querySelector("h2").textContent;

    openModal("category-delete");
}

async function confirmDeleteCategory() {
    let res = await fetch(`/api/category/${currentlyEditing.categoryID}`, {
        method: "DELETE",
    });

    if (res.status === 200) {
        let categoryEl = document.getElementById(
            `${currentlyEditing.categoryID}_category`
        );
        // get the next element and remove it (its the link grid)
        let nextEl = categoryEl.nextElementSibling;
        nextEl.remove();
        categoryEl.remove();

        closeModal();
        currentlyEditing = {};
    }
}

function roundToNearestHundredth(num) {
    return Math.round(num * 100) / 100;
}

const stylesToCopy = [
    "font-family",
    "font-size",
    "font-weight",
    "font-style",
    "color",
    "line-height",
    "letter-spacing",
    "text-transform",
    "text-align",
];

let _textMeasuringSpan,
    _textMeasuringDiv = null;

/**
 * @typedef {Object} ResizeableTextareaOptions
 * @property {HTMLElement} targetEl The element to replace.
 * @property {boolean} [fill=true] Whether to make the textarea fill the available space, or grow with the text inside.
 */

/**
 * Replaces an element with a resizable textarea containing the same text.
 * @param {ResizeableTextareaOptions[]} targetEls The elements to replace.
 * @returns (() => void) A cleanup function to remove event listeners
 */
function replaceWithResizableTextarea(targetEls) {
    let startTime = performance.now();

    /**
     * @typedef {Object} TargetInfo
     * @property {HTMLElement} targetEl The element to replace.
     * @property {boolean} fill Whether to make the textarea fill the available space, or grow with the text inside.
     * @property {string} originalText The original text of the element
     * @property {CSSStyleDeclaration} computedStyle The computed style of the element
     * @property {DOMRect} boundingRect The bounding rect of the element
     * @property {number} borderWidth The border width of the element
     * @property {number} borderHeight The border height of the element
     * @property {number} maxWidth The maximum width of the element
     */

    /**
     *  @type {TargetInfo[]}
     */
    let targetInfos = [];

    targetEls.forEach((target) => {
        let targetEl = target.targetEl;
        let fill = target.fill === undefined ? true : target.fill;
        // step 1: batch reads
        const originalText = targetEl.textContent;
        const computedStyle = window.getComputedStyle(targetEl);
        const boundingRect = targetEl.getBoundingClientRect();
        const parentBoundingRect =
            targetEl.parentElement.getBoundingClientRect();

        const borderWidth =
            parseFloat(computedStyle.borderLeftWidth) +
            parseFloat(computedStyle.borderRightWidth);
        const borderHeight =
            parseFloat(computedStyle.borderTopWidth) +
            parseFloat(computedStyle.borderBottomWidth);

        let maxWidth = parentBoundingRect.width - borderWidth;
        // take care of category headers specifically because the parent bounding box contains two other elements
        if (targetEl.tagName === "H2") {
            let imageWidth =
                targetEl.previousElementSibling.getBoundingClientRect().width;
            let actionButtonWidth =
                targetEl.nextElementSibling.getBoundingClientRect().width;

            maxWidth -= imageWidth + actionButtonWidth;
        }

        maxWidth = roundToNearestHundredth(maxWidth);

        targetInfos.push({
            targetEl,
            fill,
            originalText,
            computedStyle,
            boundingRect,
            borderWidth,
            borderHeight,
            maxWidth,
        });
    });

    const caretBuffer = 10;

    // step 2: calculate styles
    let elsInitialStyles = [];

    targetInfos.forEach((targetInfo) => {
        let fill = targetInfo.fill;

        let initialStyles = {};
        initialStyles.width = "";
        initialStyles.height = `${parseFloat(
            roundToNearestHundredth(targetInfo.boundingRect.height)
        )}px`;
        if (fill) {
            initialStyles.width = `100%`;
        } else {
            if (!_textMeasuringSpan) {
                _textMeasuringSpan = document.createElement("span");
                // Keep it off-screen and static once appended
                Object.assign(_textMeasuringSpan.style, {
                    position: "absolute",
                    left: "-9999px",
                    top: "0",
                    visibility: "hidden",
                    whiteSpace: "nowrap",
                });
                document.body.appendChild(_textMeasuringSpan);
            }

            stylesToCopy.forEach((prop) => {
                _textMeasuringSpan.style[prop] = targetInfo.computedStyle[prop];
            });

            _textMeasuringSpan.textContent =
                targetInfo.originalText === ""
                    ? targetInfo.boundingRect.placeholder || "W"
                    : targetInfo.originalText;

            let measuredTextWidth = roundToNearestHundredth(
                _textMeasuringSpan.getBoundingClientRect().width
            );

            let finalWidth = Math.min(
                measuredTextWidth + caretBuffer,
                targetInfo.maxWidth
            );
            initialStyles.width = `${finalWidth}px`;
        }

        elsInitialStyles.push({
            originalText: targetInfo.originalText,
            targetEl: targetInfo.targetEl,
            targetElComputedStyle: targetInfo.computedStyle,
            fill: fill,
            initialStyles,
        });
    });

    // step 3: batch writes
    let inputElements = [];

    elsInitialStyles.forEach((elInfo) => {
        const inputElement = document.createElement("textarea");
        inputElement.value = elInfo.originalText;
        inputElement.className = "resizable-input";
        inputElement.placeholder = elInfo.targetEl.dataset.placeholder;
        inputElement.dataset.originalElementType = elInfo.targetEl.tagName;
        inputElement.dataset.originalClassName = elInfo.targetEl.className;

        let computedStyles = {};
        // Apply inherited styles
        stylesToCopy.forEach((prop) => {
            computedStyles[prop] = elInfo.targetElComputedStyle[prop];
        });

        // Apply custom styles and calculated dimensions
        Object.assign(inputElement.style, {
            backgroundColor: "var(--color-base)",
            border: `1px solid var(--color-highlight-sm)`,
            borderRadius: "0.375rem",
            resize: "none",
            overflow: "hidden",
            outline: "none",
            ...computedStyles, // Apply calculated width and height
            ...elInfo.initialStyles, // Apply calculated width and height
        });

        inputElement.setAttribute(
            "maxlength",
            elInfo.targetEl.tagName[0] === "H" ? 50 : 150
        );

        inputElements.push({
            targetEl: elInfo.targetEl,
            fill: elInfo.fill,
            element: inputElement,
        });
    });

    function resize(inputElement, fill = false) {
        const currentInputComputedStyle = window.getComputedStyle(inputElement);
        const currentInputBorderWidth =
            parseFloat(currentInputComputedStyle.borderLeftWidth) +
            parseFloat(currentInputComputedStyle.borderRightWidth);

        const currentParentElBoundingRectWidth =
            inputElement.parentElement.getBoundingClientRect().width;

        let maxWidth = roundToNearestHundredth(
            currentParentElBoundingRectWidth
        );

        // is it maybe a bit of some math that doesnt entirely make sense to me? you bet. But does it work? Hell yeah it does
        if (inputElement.dataset.originalElementType === "H2") {
            let imageWidth =
                inputElement.previousElementSibling.getBoundingClientRect()
                    .width;
            let actionButtonWidth =
                inputElement.nextElementSibling.getBoundingClientRect().width;

            // the brain cells rub together and this vaguely makes sense to me I think but I cant explain it
            maxWidth -= imageWidth + actionButtonWidth + caretBuffer;
            maxWidth += currentInputBorderWidth;
        }

        let currentContentWidth;

        if (!fill) {
            if (!_textMeasuringSpan) {
                // Should already be created, but just in case
                _textMeasuringSpan = document.createElement("span");
                Object.assign(_textMeasuringSpan.style, {
                    position: "absolute",
                    left: "-9999px",
                    top: "0",
                    visibility: "hidden",
                    whiteSpace: "nowrap",
                });
                document.body.appendChild(_textMeasuringSpan);
            }

            stylesToCopy.forEach((prop) => {
                _textMeasuringSpan.style[prop] =
                    currentInputComputedStyle[prop];
            });

            _textMeasuringSpan.textContent =
                inputElement.value === ""
                    ? inputElement.placeholder || "W"
                    : inputElement.value;

            let measuredTextWidth =
                _textMeasuringSpan.getBoundingClientRect().width;

            currentContentWidth = Math.min(
                roundToNearestHundredth(
                    measuredTextWidth + currentInputBorderWidth
                ) + caretBuffer,
                maxWidth
            );
        } else {
            // if fill is true, width is flexible, but for measuring we need to know the *actual* width of the content
            currentContentWidth = maxWidth;
        }

        if (!_textMeasuringDiv) {
            _textMeasuringDiv = document.createElement("div");
            Object.assign(_textMeasuringDiv.style, {
                position: "absolute",
                left: "-9999px",
                top: "0",
                visibility: "hidden",
                // Allow wrapping exactly like a textarea
                whiteSpace: "pre-wrap",
                wordWrap: "break-word",
            });
            document.body.appendChild(_textMeasuringDiv);
        }

        [
            "borderLeftWidth",
            "borderRightWidth",
            "borderTopWidth",
            "borderBottomWidth",
            ...stylesToCopy,
        ].forEach((prop) => {
            _textMeasuringDiv.style[prop] = currentInputComputedStyle[prop];
        });

        _textMeasuringDiv.style.width = `${currentContentWidth}px`;
        _textMeasuringDiv.textContent =
            inputElement.value === ""
                ? inputElement.placeholder || "W"
                : inputElement.value;
        let measuredContentHeight =
            _textMeasuringDiv.getBoundingClientRect().height;

        // we set the height = 0 so that if a row is deleted, the height will be recalculated correctly
        inputElement.style.width = `${currentContentWidth}px`;
        inputElement.style.height = "0px";
        inputElement.style.height = `${measuredContentHeight}px`;
    }

    function resizeAll() {
        inputElements.forEach((inputEl) => {
            resize(inputEl.element, inputEl.fill);
        });
    }

    // step 4: append
    inputElements.forEach((inputEl) => {
        inputEl.targetEl.parentNode.replaceChild(
            inputEl.element,
            inputEl.targetEl
        );
        inputEl.element.addEventListener("input", () => {
            resize(inputEl.element, inputEl.fill);
        });
    });

    let resizeScheduled = false;

    function windowResize() {
        if (!resizeScheduled) {
            resizeScheduled = true;
            requestAnimationFrame(() => {
                resizeAll();
                resizeScheduled = false;
            });
        }
    }

    window.addEventListener("resize", windowResize);

    // if the caller wants to focus the textarea, they can do it themselves

    return () => {
        window.removeEventListener("resize", windowResize);
    };
}

/**
 * Restores an element from a textarea
 * @param {HTMLElement} inputEl The textarea to restore
 * @param {string} originalText The original text of the textarea
 */
function restoreElementFromInput(inputEl, originalText) {
    const computedStyle = window.getComputedStyle(inputEl);
    let styles = {};

    let elementType = inputEl.dataset.originalElementType;
    const newElement = document.createElement(elementType);
    newElement.textContent = originalText;
    newElement.className = inputEl.dataset.originalClassName;
    newElement.dataset.placeholder = inputEl.placeholder;

    stylesToCopy.forEach((prop) => {
        styles[prop] = computedStyle[prop];
    });

    Object.assign(newElement.style, {
        ...styles,
        border: "1px solid #0000",
    });

    inputEl.parentNode.replaceChild(newElement, inputEl);
}
