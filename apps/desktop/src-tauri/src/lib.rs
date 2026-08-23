mod desktop;
#[cfg(not(dev))]
mod frontend;
mod tray;

pub use desktop::run;
