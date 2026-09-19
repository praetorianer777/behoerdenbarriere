import { screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Statistics } from "./Statistics";
import { api } from "../api/client";
import { usage } from "../test/fixtures";
import { renderPage } from "../test/render";

describe("Statistik", () => {
	beforeEach(() => {
		vi.spyOn(api, "usage").mockResolvedValue(usage);
	});
	afterEach(() => vi.restoreAllMocks());

	it("zeigt Besuche und Seitenaufrufe", async () => {
		renderPage(<Statistics />);

		await screen.findByRole("heading", {
			level: 1,
			name: "Nutzung dieser Seite",
		});
		expect(screen.getByText("31")).toBeInTheDocument();
		expect(screen.getByText("84")).toBeInTheDocument();
	});

	it("benennt die Seiten verständlich statt mit ihrem Pfad", async () => {
		renderPage(<Statistics />);

		const row = await screen.findByRole("row", { name: /Behördenseiten/ });
		expect(within(row).getByText("24")).toBeInTheDocument();
		expect(screen.queryByText("/behoerde/:slug")).not.toBeInTheDocument();
	});

	// Die Seite ist die Selbstauskunft des Projekts: was nicht gespeichert wird, muss
	// dort auch stehen.
	it("erklärt, was nicht gespeichert wird", async () => {
		renderPage(<Statistics />);

		await screen.findByRole("heading", { level: 2, name: "Wie gezählt wird" });
		expect(screen.getByText(/keine Cookies/)).toBeInTheDocument();
		expect(
			screen.getByText(/Nicht gespeichert werden: IP-Adressen/),
		).toBeInTheDocument();
	});

	it("meldet einen leeren Zeitraum, statt Nullen zu zeigen", async () => {
		vi.spyOn(api, "usage").mockResolvedValue({
			...usage,
			days: [],
			pages: [],
			agencies: [],
			endpoints: [],
		});
		renderPage(<Statistics />);

		expect(
			await screen.findByText(
				"Für diesen Zeitraum liegen noch keine Zahlen vor.",
			),
		).toBeInTheDocument();
	});
});
