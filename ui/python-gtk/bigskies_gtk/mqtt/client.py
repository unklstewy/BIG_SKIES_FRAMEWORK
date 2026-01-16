"""
MQTT client wrapper for BigSkies framework.
Handles connection, subscription, and message publishing with BigSkies topic structure.
"""
import json
import logging
import uuid
from typing import Callable, Optional, Dict, Any
import paho.mqtt.client as mqtt
from gi.repository import GLib

logger = logging.getLogger(__name__)


class BigSkiesMQTTClient:
    """MQTT client wrapper for BigSkies framework communication."""
    
    def __init__(self, broker_host: str = "localhost", broker_port: int = 1883):
        """
        Initialize MQTT client.
        
        Args:
            broker_host: MQTT broker hostname
            broker_port: MQTT broker port
        """
        self.broker_host = broker_host
        self.broker_port = broker_port
        self.client_id = f"bigskies-gtk-{uuid.uuid4().hex[:8]}"
        self.client = mqtt.Client(client_id=self.client_id)
        self.client.on_connect = self._on_connect
        self.client.on_disconnect = self._on_disconnect
        self.client.on_message = self._on_message
        
        self.connected = False
        self.message_handlers: Dict[str, list] = {}
        self.connect_callback: Optional[Callable] = None
        self.disconnect_callback: Optional[Callable] = None
        
    def set_callbacks(self, 
                     on_connect: Optional[Callable] = None,
                     on_disconnect: Optional[Callable] = None):
        """Set connection status callbacks."""
        self.connect_callback = on_connect
        self.disconnect_callback = on_disconnect
        
    def connect(self) -> bool:
        """
        Connect to MQTT broker.
        
        Returns:
            True if connection initiated successfully
        """
        try:
            logger.info(f"Connecting to MQTT broker at {self.broker_host}:{self.broker_port}")
            self.client.connect(self.broker_host, self.broker_port, 60)
            self.client.loop_start()
            return True
        except Exception as e:
            logger.error(f"Failed to connect to MQTT broker: {e}")
            return False
            
    def disconnect(self):
        """Disconnect from MQTT broker."""
        logger.info("Disconnecting from MQTT broker")
        self.client.loop_stop()
        self.client.disconnect()
        
    def subscribe(self, topic: str, callback: Callable[[str, Dict[str, Any]], None]):
        """
        Subscribe to MQTT topic with callback.
        
        Args:
            topic: MQTT topic to subscribe to
            callback: Function to call when message received (topic, payload_dict)
        """
        if topic not in self.message_handlers:
            self.message_handlers[topic] = []
            self.client.subscribe(topic)
            logger.info(f"Subscribed to topic: {topic}")
        
        self.message_handlers[topic].append(callback)
        
    def publish(self, topic: str, payload: Dict[str, Any], qos: int = 1):
        """
        Publish message to MQTT topic.
        
        Args:
            topic: MQTT topic
            payload: Message payload as dictionary
            qos: Quality of service level
        """
        try:
            payload_str = json.dumps(payload)
            self.client.publish(topic, payload_str, qos=qos)
            logger.debug(f"Published to {topic}: {payload_str}")
        except Exception as e:
            logger.error(f"Failed to publish to {topic}: {e}")
            
    def _on_connect(self, client, userdata, flags, rc):
        """Handle MQTT connection."""
        if rc == 0:
            self.connected = True
            logger.info("Connected to MQTT broker")
            
            # Resubscribe to all topics
            for topic in self.message_handlers.keys():
                client.subscribe(topic)
                
            # Call connect callback on main thread
            if self.connect_callback:
                GLib.idle_add(self.connect_callback)
        else:
            logger.error(f"Failed to connect, return code {rc}")
            
    def _on_disconnect(self, client, userdata, rc):
        """Handle MQTT disconnection."""
        self.connected = False
        logger.warning(f"Disconnected from MQTT broker, code {rc}")
        
        # Call disconnect callback on main thread
        if self.disconnect_callback:
            GLib.idle_add(self.disconnect_callback)
            
    def _on_message(self, client, userdata, msg):
        """Handle incoming MQTT message."""
        try:
            # Decode payload
            payload_str = msg.payload.decode('utf-8')
            payload_dict = json.loads(payload_str) if payload_str else {}
            
            logger.debug(f"Received message on {msg.topic}")
            
            # Find matching handlers
            for topic_pattern, handlers in self.message_handlers.items():
                if self._topic_matches(msg.topic, topic_pattern):
                    for handler in handlers:
                        # Call handler on main thread
                        GLib.idle_add(handler, msg.topic, payload_dict)
                        
        except json.JSONDecodeError as e:
            logger.error(f"Failed to decode JSON from {msg.topic}: {e}")
        except Exception as e:
            logger.error(f"Error handling message from {msg.topic}: {e}")
            
    @staticmethod
    def _topic_matches(topic: str, pattern: str) -> bool:
        """
        Check if topic matches pattern (supporting MQTT wildcards).
        
        Args:
            topic: Actual topic
            pattern: Pattern with + (single level) or # (multi level) wildcards
            
        Returns:
            True if topic matches pattern
        """
        if pattern == topic:
            return True
            
        topic_parts = topic.split('/')
        pattern_parts = pattern.split('/')
        
        # # must be last and matches everything after
        if '#' in pattern_parts:
            hash_idx = pattern_parts.index('#')
            if hash_idx != len(pattern_parts) - 1:
                return False
            pattern_parts = pattern_parts[:hash_idx]
            topic_parts = topic_parts[:hash_idx]
            
        if len(topic_parts) != len(pattern_parts):
            return False
            
        for topic_part, pattern_part in zip(topic_parts, pattern_parts):
            if pattern_part != '+' and pattern_part != topic_part:
                return False
                
        return True
