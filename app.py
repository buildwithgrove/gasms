from pathlib import Path
from textual.app import App, ComposeResult
from textual.widgets import Static, DataTable, Input, Footer
from textual.binding import Binding
from textual.containers import Horizontal, Container
from pocket import load_config, query_application


# === Helpers to load ASCII art ===
def load_ascii_art(filename: str) -> str:
    return Path(f"art/{filename}").read_text()


LOGO_LINE = load_ascii_art("logo.txt").splitlines()[0]
SPLASH_ART = load_ascii_art("splash.txt")


# === Custom Header Bar (like k9s) ===
class HeaderBar(Horizontal):
    def compose(self) -> ComposeResult:
        yield Static("r:Refresh  :Command", id="commands")
        yield Static(LOGO_LINE, id="logo")


# === Main App ===
class GASMSApp(App):
    CSS_PATH = "app.css"

    BINDINGS = [
        Binding("r", "refresh", "Refresh"),
        Binding(":", "enter_command_mode", "Command")
    ]

    def compose(self) -> ComposeResult:
        yield HeaderBar()

        self.loading_screen = Static(SPLASH_ART, id="splash", expand=True)
        self.table = DataTable()
        self.table.add_columns("App Address", "Stake (POKT)", "Service ID", "Gateway")

        self.container = Container(self.loading_screen, self.table)
        yield self.container

        self.cmd = Input(placeholder="Command (e.g. :q)", id="cmd_input")
        self.cmd.display = False
        yield self.cmd

        yield Footer()

    async def on_mount(self):
        print("🚀 on_mount triggered")
        self.config = load_config()
        self.rpc = self.config["rpc_endpoint"]
        self.gateway = self.config["gateways"][0]

        # Wait one tick so splash is rendered
        self.call_later(self.finish_boot)

    async def finish_boot(self):
        await self.loading_screen.remove()
        self.refresh_table()

    def action_refresh(self):
        self.refresh_table()

    def refresh_table(self):
        print("🔁 Refreshing table...")
        self.table.clear()

        for app in self.config["applications"]:
            print(f"🔍 Querying {app}")
            data = query_application(app, self.rpc)
            print(f"↩️ Response: {data}")
            if "error" in data:
                self.table.add_row(app, "Error", "-", "-")
                continue

            try:
                stake_amt = round(int(data["stake"]["amount"]) / 1_000_000, 2)
                service_id = data["service_configs"][0]["service_id"] if data["service_configs"] else "-"
                self.table.add_row(app, f"{stake_amt:.2f}", service_id, self.gateway)
            except Exception as e:
                self.table.add_row(app, f"ParseErr: {e}", "-", "-")

    def action_enter_command_mode(self):
        self.cmd.display = True
        self.cmd.focus()

    def on_input_submitted(self, event: Input.Submitted):
        cmd = event.value.strip()
        print(f"🔧 Command received: {cmd}")
        self.cmd.display = False
        self.refresh()

        if cmd == "q":
            self.app.exit()

    def on_input_blurred(self, event: Input.Blurred):
        self.cmd.display = False
        self.refresh()


# === Entry Point ===
if __name__ == "__main__":
    print("🏁 Running GASMSApp...")
    GASMSApp().run()

