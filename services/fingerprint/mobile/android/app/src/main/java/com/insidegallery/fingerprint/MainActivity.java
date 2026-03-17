package com.insidegallery.fingerprint;

import android.os.Bundle;

import com.insidegallery.fingerprint.mobile.EbitenView;
import com.insidegallery.fingerprint.mobile.Mobile;

import android.app.Activity;
import android.widget.FrameLayout;

public class MainActivity extends Activity {
    private EbitenView ebitenView;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        // Set writable storage directory for game saves
        Mobile.setStorageDir(getFilesDir().getAbsolutePath());

        FrameLayout layout = new FrameLayout(this);
        ebitenView = new EbitenView(this);
        layout.addView(ebitenView);
        setContentView(layout);
    }

    @Override
    protected void onPause() {
        super.onPause();
        ebitenView.suspendGame();
    }

    @Override
    protected void onResume() {
        super.onResume();
        ebitenView.resumeGame();
    }
}
