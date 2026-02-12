import { test, expect } from "@playwright/test";

test.describe("Step 1: Source Connection", () => {
  test.beforeEach(async ({ request }) => {
    await request.put("/api/state/step", {
      data: { step: "source_connection" },
    });
  });

  test("page loads with pre-filled connection form", async ({ page }) => {
    await page.goto("/source");
    await expect(page.getByRole("heading", { name: "Source Connection" })).toBeVisible();
    await expect(page.getByText("Host")).toBeVisible();
    await expect(page.getByText("Port")).toBeVisible();
    await expect(page.getByText("Database", { exact: true })).toBeVisible();
  });

  test("test connection shows success", async ({ page }) => {
    await page.goto("/source");
    await page.getByRole("button", { name: "Test Connection" }).click();
    await expect(page.getByText("Connection successful")).toBeVisible({ timeout: 10000 });
  });

  test("discover schema shows table count and continue button", async ({ page }) => {
    await page.goto("/source");

    await page.getByRole("button", { name: "Test Connection" }).click();
    await expect(page.getByText("Connection successful")).toBeVisible({ timeout: 10000 });

    await page.getByRole("button", { name: "Discover Schema" }).click();
    await expect(page.getByText(/Discovered \d+ tables/)).toBeVisible({ timeout: 15000 });

    await expect(page.getByRole("button", { name: /Continue to Table Selection/ })).toBeVisible();
  });

  test("continue navigates to table selection", async ({ page }) => {
    await page.goto("/source");

    await page.getByRole("button", { name: "Test Connection" }).click();
    await expect(page.getByText("Connection successful")).toBeVisible({ timeout: 10000 });

    await page.getByRole("button", { name: "Discover Schema" }).click();
    await expect(page.getByText(/Discovered \d+ tables/)).toBeVisible({ timeout: 15000 });

    await page.getByRole("button", { name: /Continue to Table Selection/ }).click();
    await expect(page).toHaveURL(/\/tables/, { timeout: 5000 });
    await expect(page.getByRole("heading", { name: "Table Selection" })).toBeVisible();
  });
});

test.describe("Step 2: Table Selection", () => {
  test.beforeEach(async ({ request }) => {
    await request.post("/api/source/discover", {
      data: {
        type: "postgresql",
        host: "postgres",
        port: 5432,
        database: "pagila",
        username: "reloquent",
        password: "reloquent",
        ssl: false,
      },
    });
    await request.put("/api/state/step", {
      data: { step: "table_selection" },
    });
  });

  test("shows discovered tables", async ({ page }) => {
    await page.goto("/tables");
    await expect(page.getByRole("heading", { name: "Table Selection" })).toBeVisible();
    await expect(page.getByRole("cell", { name: "actor", exact: true })).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole("cell", { name: "film", exact: true })).toBeVisible();
  });
});

test.describe("Step 3: Denormalization Design", () => {
  async function setupDesignStep(request: any) {
    await request.post("/api/source/discover", {
      data: {
        type: "postgresql",
        host: "postgres",
        port: 5432,
        database: "pagila",
        username: "reloquent",
        password: "reloquent",
        ssl: false,
      },
    });
    await request.post("/api/tables/select", {
      data: { tables: ["actor", "film", "film_actor", "category", "film_category"] },
    });
    await request.put("/api/state/step", {
      data: { step: "denormalization" },
    });
  }

  test("loads with canvas and nodes", async ({ page, request }) => {
    await setupDesignStep(request);
    await page.goto("/design");
    await page.waitForLoadState("networkidle");
    await expect(page.getByRole("heading", { name: "Denormalization Design" })).toBeVisible({ timeout: 15000 });
    await expect(page.locator(".react-flow")).toBeVisible({ timeout: 10000 });
    // Nodes should render (table nodes are visible within the viewport)
    await expect(page.locator(".react-flow__node").first()).toBeVisible({ timeout: 5000 });
  });

  test("edges exist in the canvas", async ({ page, request }) => {
    await setupDesignStep(request);
    await page.goto("/design");
    await page.waitForLoadState("networkidle");
    await expect(page.locator(".react-flow")).toBeVisible({ timeout: 15000 });
    // React Flow SVG edges may be outside viewport, so check count instead of visibility
    await expect(page.locator(".react-flow__edge")).not.toHaveCount(0, { timeout: 5000 });
  });

  test("clicking edge opens config panel with radio buttons", async ({ page, request }) => {
    await setupDesignStep(request);
    await page.goto("/design");
    await page.waitForLoadState("networkidle");
    await expect(page.locator(".react-flow")).toBeVisible({ timeout: 15000 });

    // React Flow SVG edges may be outside the visible viewport
    // Use force click on the first edge element
    const edge = page.locator(".react-flow__edge").first();
    await expect(edge).toHaveCount(1, { timeout: 5000 });
    await edge.click({ force: true });

    // Config panel should appear
    await expect(page.getByText("Embed as array")).toBeVisible({ timeout: 3000 });
    await expect(page.getByText("Embed as single")).toBeVisible();
    await expect(page.getByText("Reference")).toBeVisible();
    await expect(page.getByRole("button", { name: "Remove Relationship" })).toBeVisible();
  });

  test("changing relationship type updates radio selection", async ({ page, request }) => {
    await setupDesignStep(request);
    await page.goto("/design");
    await page.waitForLoadState("networkidle");
    await expect(page.locator(".react-flow")).toBeVisible({ timeout: 15000 });

    const edge = page.locator(".react-flow__edge").first();
    await expect(edge).toHaveCount(1, { timeout: 5000 });
    await edge.click({ force: true });
    await expect(page.getByText("Embed as array")).toBeVisible({ timeout: 3000 });

    const arrayRadio = page.locator('input[type="radio"][value="array"]');
    const singleRadio = page.locator('input[type="radio"][value="single"]');
    const refRadio = page.locator('input[type="radio"][value="reference"]');

    // Click "Reference"
    await refRadio.click();
    await expect(refRadio).toBeChecked();
    await expect(arrayRadio).not.toBeChecked();

    // Click "Embed as array"
    await arrayRadio.click();
    await expect(arrayRadio).toBeChecked();
    await expect(refRadio).not.toBeChecked();

    // Click "Embed as single"
    await singleRadio.click();
    await expect(singleRadio).toBeChecked();
    await expect(arrayRadio).not.toBeChecked();
  });

  test("remove relationship closes config panel", async ({ page, request }) => {
    await setupDesignStep(request);
    await page.goto("/design");
    await page.waitForLoadState("networkidle");
    await expect(page.locator(".react-flow")).toBeVisible({ timeout: 15000 });

    const edge = page.locator(".react-flow__edge").first();
    await expect(edge).toHaveCount(1, { timeout: 5000 });
    await edge.click({ force: true });
    await expect(page.getByRole("button", { name: "Remove Relationship" })).toBeVisible({ timeout: 3000 });

    await page.getByRole("button", { name: "Remove Relationship" }).click();
    await expect(page.getByRole("button", { name: "Remove Relationship" })).not.toBeVisible();
  });

  test("save and continue navigates to type mapping", async ({ page, request }) => {
    await setupDesignStep(request);
    await page.goto("/design");
    await page.waitForLoadState("networkidle");
    await expect(page.locator(".react-flow")).toBeVisible({ timeout: 15000 });

    await page.getByRole("button", { name: /Save & Continue/ }).click();
    await expect(page).toHaveURL(/\/types/, { timeout: 5000 });
  });
});

test.describe("Sidebar Navigation", () => {
  test("completed step shows checkmark when not current", async ({ page, request }) => {
    await request.post("/api/source/discover", {
      data: {
        type: "postgresql",
        host: "postgres",
        port: 5432,
        database: "pagila",
        username: "reloquent",
        password: "reloquent",
        ssl: false,
      },
    });
    // Move to table_selection so source_connection is completed but not current
    await request.put("/api/state/step", {
      data: { step: "table_selection" },
    });

    await page.goto("/tables");
    await page.waitForLoadState("networkidle");

    const sidebar = page.locator("aside");
    const sourceLink = sidebar.locator("a", { hasText: "Source Connection" });
    await expect(sourceLink).toBeVisible({ timeout: 3000 });
    // Should show checkmark for completed step
    await expect(sourceLink).toContainText("✓", { timeout: 3000 });
  });

  test("can navigate backward to completed steps", async ({ page, request }) => {
    await request.post("/api/source/discover", {
      data: {
        type: "postgresql",
        host: "postgres",
        port: 5432,
        database: "pagila",
        username: "reloquent",
        password: "reloquent",
        ssl: false,
      },
    });
    await request.put("/api/state/step", {
      data: { step: "table_selection" },
    });

    await page.goto("/tables");
    await expect(page.getByRole("heading", { name: "Table Selection" })).toBeVisible();

    const sidebar = page.locator("aside");
    await sidebar.getByText("Source Connection").click();
    await expect(page).toHaveURL(/\/source/, { timeout: 3000 });
  });
});
