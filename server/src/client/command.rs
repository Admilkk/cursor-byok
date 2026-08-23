use crate::model::{CanonicalMessage, RuntimeEvent, ToolResult};

#[derive(Clone, Debug, PartialEq)]
pub enum ClientCommand {
    ToolResult(ToolResult),
    RuntimeMessage(CanonicalMessage),
    RuntimeEvent(RuntimeEvent),
    ClientClosed { error: String },
    Cancel,
}
