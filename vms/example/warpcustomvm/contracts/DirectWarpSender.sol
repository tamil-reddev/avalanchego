// SPDX-License-Identifier: MIT
pragma solidity ^0.8.18;

/**
 * @title DirectWarpSender
 * @notice Sends Warp messages directly from C-Chain to custom VMs
 * @dev Use this contract to send messages to custom VMs that don't have Teleporter
 *      This uses the Warp precompile directly instead of Teleporter
 */

// Warp Precompile Interface
interface IWarpMessenger {
    event SendWarpMessage(bytes message);
    
    function sendWarpMessage(bytes calldata payload) external returns (bytes32 messageID);
    function getBlockchainID() external view returns (bytes32);
}

contract DirectWarpSender {
    // Warp Precompile address (same on all Avalanche chains)
    IWarpMessenger public constant WARP_PRECOMPILE = 
        IWarpMessenger(0x0200000000000000000000000000000000000005);
    
    // Events
    event WarpMessageSent(
        bytes32 indexed messageID,
        bytes32 indexed destinationBlockchainID,
        address indexed sender,
        bytes payload
    );
    
    /**
     * @notice Send a simple text message to a custom VM
     * @param destinationBlockchainID The blockchain ID of the custom VM (for tracking only)
     * @param message The text message to send
     * @return messageID The Warp message ID
     */
    function sendTextMessage(
        bytes32 destinationBlockchainID,
        string calldata message
    ) external returns (bytes32 messageID) {
        // Encode the message as bytes
        bytes memory payload = abi.encode(message);
        
        // Send via Warp precompile
        messageID = WARP_PRECOMPILE.sendWarpMessage(payload);
        
        emit WarpMessageSent(
            messageID,
            destinationBlockchainID,
            msg.sender,
            payload
        );
    }
    
    /**
     * @notice Send raw bytes to a custom VM
     * @param destinationBlockchainID The blockchain ID of the custom VM (for tracking only)
     * @param payload The raw bytes to send
     * @return messageID The Warp message ID
     */
    function sendRawMessage(
        bytes32 destinationBlockchainID,
        bytes calldata payload
    ) external returns (bytes32 messageID) {
        // Send directly via Warp precompile
        messageID = WARP_PRECOMPILE.sendWarpMessage(payload);
        
        emit WarpMessageSent(
            messageID,
            destinationBlockchainID,
            msg.sender,
            payload
        );
    }
    
    /**
     * @notice Get the current blockchain ID
     * @return The blockchain ID of this chain (C-Chain)
     */
    function getBlockchainID() external view returns (bytes32) {
        return WARP_PRECOMPILE.getBlockchainID();
    }
}
